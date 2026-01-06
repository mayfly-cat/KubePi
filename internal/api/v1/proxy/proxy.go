package proxy

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/KubeOperator/kubepi/internal/api/v1/session"
	v1Cluster "github.com/KubeOperator/kubepi/internal/model/v1/cluster"
	"github.com/KubeOperator/kubepi/internal/service/v1/cluster"
	"github.com/KubeOperator/kubepi/internal/service/v1/clusterbinding"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	ingresshistory "github.com/KubeOperator/kubepi/internal/service/v1/ingresshistory"
	pkgV1 "github.com/KubeOperator/kubepi/pkg/api/v1"
	"github.com/KubeOperator/kubepi/pkg/kubernetes"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type Handler struct {
	clusterService        cluster.Service
	clusterBindingService clusterbinding.Service
}

func NewHandler() *Handler {
	return &Handler{
		clusterService:        cluster.NewService(),
		clusterBindingService: clusterbinding.NewService(),
	}
}

// parseIngressInfo 解析 Ingress 路径信息
func (h *Handler) parseIngressInfo(proxyPath string) (namespace, ingressName string, ok bool) {
	if !strings.Contains(proxyPath, "ingresses") {
		return "", "", false
	}

	parts := strings.Split(proxyPath, "/")
	for i, part := range parts {
		if part == "namespaces" && i+1 < len(parts) {
			namespace = parts[i+1]
		}
		if part == "ingresses" && i+1 < len(parts) {
			ingressName = parts[i+1]
			break
		}
	}

	if namespace == "" || ingressName == "" {
		return "", "", false
	}

	return namespace, ingressName, true
}

// saveIngressHistoryBeforeUpdate 在更新 Ingress 之前保存当前版本（仅当不存在历史记录时）
func (h *Handler) saveIngressHistoryBeforeUpdate(ctx *context.Context, name string, proxyPath string, profile session.UserProfile) error {
	// 检查是否是 Ingress 的 PUT 请求
	if ctx.Request().Method != http.MethodPut {
		return nil
	}

	namespace, ingressName, ok := h.parseIngressInfo(proxyPath)
	if !ok {
		return nil
	}

	// 检查是否已有历史版本
	ingressHistoryService := ingresshistory.NewService()
	latestVersion, err := ingressHistoryService.GetLatestVersion(name, namespace, ingressName, common.DBOptions{})
	if err != nil {
		// 查询失败，不保存历史
		return nil
	}

	// 如果已有历史版本，不需要在更新前保存
	if latestVersion > 0 {
		return nil
	}

	// 如果没有历史版本，需要保存当前版本作为最原始的记录
	c, err := h.clusterService.Get(name, common.DBOptions{})
	if err != nil {
		return fmt.Errorf("get cluster failed: %w", err)
	}

	ts, err := h.generateTLSTransport(c, profile)
	if err != nil {
		return fmt.Errorf("generate transport failed: %w", err)
	}

	httpClient := http.Client{Transport: ts}
	k := kubernetes.NewKubernetes(c)
	clusterVersionMinor, err := k.VersionMinor()
	if err != nil {
		return fmt.Errorf("get cluster version failed: %w", err)
	}

	// 构建获取当前 Ingress 的 URL
	getPath := proxyPath
	compatibleClusterVersion(clusterVersionMinor, &getPath)
	getUrl := fmt.Sprintf("%s%s", c.Spec.Connect.Forward.ApiServer, getPath)

	getReq, err := http.NewRequest(http.MethodGet, getUrl, nil)
	if err != nil {
		return fmt.Errorf("create get request failed: %w", err)
	}

	getResp, err := httpClient.Do(getReq)
	if err != nil {
		// 如果获取失败，不保存历史
		return nil
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		// 如果当前 Ingress 不存在，不保存历史
		return nil
	}

	currentIngressData, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return fmt.Errorf("read current ingress failed: %w", err)
	}

	// 解析当前 Ingress 数据
	var currentIngress interface{}
	if err := json.Unmarshal(currentIngressData, &currentIngress); err != nil {
		return fmt.Errorf("parse current ingress failed: %w", err)
	}

	// 保存历史版本（用于维护最原始的记录数据）
	_, err = ingressHistoryService.SaveHistory(
		name,
		namespace,
		ingressName,
		currentIngress,
		profile.Name,
		"Auto saved before update",
		common.DBOptions{},
	)

	if err != nil {
		// 保存历史失败不应该阻止更新操作，只记录错误
		return nil
	}

	return nil
}

// saveIngressHistoryAfterOperation 在创建或更新 Ingress 之后保存新版本
func (h *Handler) saveIngressHistoryAfterOperation(ctx *context.Context, name string, proxyPath string, responseBody []byte, statusCode int, profile session.UserProfile) error {
	// 检查是否是 Ingress 的 POST 或 PUT 请求，且操作成功
	if (ctx.Request().Method != http.MethodPost && ctx.Request().Method != http.MethodPut) || statusCode < 200 || statusCode >= 300 {
		return nil
	}

	// 解析响应中的 Ingress 数据
	var ingressData map[string]interface{}
	if err := json.Unmarshal(responseBody, &ingressData); err != nil {
		// 如果解析失败，不保存历史
		return nil
	}

	// 从响应数据中获取 namespace 和 name
	metadata, ok := ingressData["metadata"].(map[string]interface{})
	if !ok {
		return nil
	}

	namespace, ok := metadata["namespace"].(string)
	if !ok || namespace == "" {
		return nil
	}

	ingressName, ok := metadata["name"].(string)
	if !ok || ingressName == "" {
		return nil
	}

	// 保存历史版本
	ingressHistoryService := ingresshistory.NewService()
	description := "Auto saved after create"
	if ctx.Request().Method == http.MethodPut {
		description = "Auto saved after update"
	}

	_, err := ingressHistoryService.SaveHistory(
		name,
		namespace,
		ingressName,
		ingressData,
		profile.Name,
		description,
		common.DBOptions{},
	)

	if err != nil {
		// 保存历史失败不应该影响操作结果，只记录错误
		return nil
	}

	return nil
}

type NamespaceResourceContainer struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []interface{} `json:"items"`
	Namespaces      []string      `json:"namespaces"`
}

func (h *Handler) KubernetesAPIProxy() iris.Handler {
	return func(ctx *context.Context) {
		// 解析参数
		name := ctx.Params().GetString("name")
		proxyPath := ensureProxyPathValid(ctx.Params().GetString("p"))
		namespace := ctx.URLParam("namespace")
		keywords := ctx.URLParam("keywords")
		search := false
		if ctx.URLParamExists("search") {
			search, _ = ctx.URLParamBool("search")
		}

		requestMethod := ctx.Request().Method
		// 获取当亲集群
		c, err := h.clusterService.Get(name, common.DBOptions{})
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", fmt.Sprintf("get cluster failed: %s", err.Error()))
			return
		}
		// 获取session
		u := ctx.Values().Get("profile")
		profile := u.(session.UserProfile)
		// 生成transport
		ts, err := h.generateTLSTransport(c, profile)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		// 生成httpClient
		httpClient := http.Client{Transport: ts}
		k := kubernetes.NewKubernetes(c)
		clusterVersionMinor, err := k.VersionMinor()
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		compatibleClusterVersion(clusterVersionMinor, &proxyPath)

		//判断是否已经包含了namespace的查询
		hasNsFilter := hasNamespaceFilter(proxyPath)
		resourceName, err := parseResourceName(proxyPath)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		// 判断资源类型是否是namespace级别的
		namespaced, err := k.IsNamespacedResource(resourceName)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		if strings.Contains(proxyPath, "namespaces") {
			namespaced = false
		}
		canVisitAll := false
		if profile.IsAdministrator {
			canVisitAll = true
		} else {
			canVisitAll, err = k.CanVisitAllNamespace(profile.Name)
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err)
				return
			}
		}
		apiUrl, err := url.Parse(fmt.Sprintf("%s%s", c.Spec.Connect.Forward.ApiServer, proxyPath))
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		apiUrl.RawQuery = ctx.Request().URL.RawQuery
		if http.MethodGet == requestMethod && namespace == "" && namespaced && !canVisitAll {
			// 调用多namespace 逻辑
			allowedNamespaces, err := k.GetUserNamespaceNames(profile.Name)
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err)
				return
			}
			resp, err := fetchMultiNamespaceResource(&httpClient, allowedNamespaces, *apiUrl)
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err)
				return
			}
			klo := K8sListObj{
				Kind:       resp.Kind,
				ApiVersion: resp.APIVersion,
				Metadata:   resp.ListMeta,
				Items:      resp.Items,
			}

			p, err := pagerAndSearch(ctx, klo, keywords)
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err)
				return
			}
			_ = ctx.JSON(p)
			return
		}
		if http.MethodGet == requestMethod && namespaced && namespace != "" && !hasNsFilter {
			apiUrl.Path = addUrlNamespace(apiUrl.Path, namespace)
		}

		// 对于 PUT 请求，需要先读取请求体（可能用于保存历史）
		var requestBody []byte
		if requestMethod == http.MethodPut || requestMethod == http.MethodPost || requestMethod == http.MethodPatch {
			requestBody, _ = ctx.GetBody()
		}

		// 如果是 Ingress 的 PUT 请求，在更新之前保存当前版本
		if requestMethod == http.MethodPut {
			_ = h.saveIngressHistoryBeforeUpdate(ctx, name, proxyPath, profile)
		}

		// 重新创建请求体（因为已经读取过了）
		var bodyReader io.Reader
		if len(requestBody) > 0 {
			bodyReader = bytes.NewReader(requestBody)
		}

		req, err := http.NewRequest(ctx.Request().Method, apiUrl.String(), bodyReader)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		if ctx.Method() == "PATCH" {
			req.Header.Set("Content-Type", "application/merge-patch+json")
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", err)
			return
		}
		rawResp, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusForbidden {
			resp.StatusCode = http.StatusInternalServerError
		}

		// 如果是 Ingress 的 POST 或 PUT 请求，在操作成功后保存新版本
		if requestMethod == http.MethodPost || requestMethod == http.MethodPut {
			_ = h.saveIngressHistoryAfterOperation(ctx, name, proxyPath, rawResp, resp.StatusCode, profile)
		}
		if req.Method == http.MethodGet && search {
			var listObj K8sListObj
			if err := json.Unmarshal(rawResp, &listObj); err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err)
				return
			}
			p, err := pagerAndSearch(ctx, listObj, keywords)
			if err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Values().Set("message", err.Error())
				return
			}
			_ = ctx.JSON(p)
			return
		}
		ctx.StatusCode(resp.StatusCode)
		ctx.Values().Set("message", string(rawResp))
		_, _ = ctx.Write(rawResp)
	}
}

var timeTemplate = "2006-01-02T15:04:05Z"

func pagerAndSearch(ctx *context.Context, listObj K8sListObj, keywords string) (*pkgV1.Page, error) {
	num, err1 := ctx.Values().GetInt("pageNum")
	size, err2 := ctx.Values().GetInt("pageSize")
	var p pkgV1.Page
	if listObj.Kind != "NodeList" {
		listObj.Sort()
	}
	if keywords != "" {
		listObj.Items = fieldFilter(listObj.Items, withNamespaceAndNameMatcher(keywords))
	}
	if err1 == nil && err2 == nil {
		tt, items, err := pageFilter(num, size, listObj.Items)
		if err != nil {
			return nil, err
		}
		listObj.Items = items
		p.Total = tt
		p.Items = listObj.Items
		return &p, nil
	}
	p.Total = len(listObj.Items)
	p.Items = listObj.Items
	return &p, nil
}

func getTime(obj interface{}) time.Time {
	//判断是否存在lasttime
	o := obj.(map[string]interface{})
	if o["lastTimestamp"] != nil {
		strTime := o["lastTimestamp"].(string)
		t, err := time.ParseInLocation(timeTemplate, strTime, time.Local)
		if err != nil {
			return time.Time{}
		}
		return t
	}
	if o["metadata"].(map[string]interface{}) != nil {
		metadata := o["metadata"].(map[string]interface{})
		if metadata["creationTimestamp"] != nil {
			strTime := metadata["creationTimestamp"].(string)
			t, err := time.ParseInLocation(timeTemplate, strTime, time.Local)
			if err != nil {
				return time.Time{}
			}
			return t
		}
	}
	return time.Time{}
}

type ItemList []interface{}

type Item struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
}

func (a ItemList) Len() int {
	return len(a)
}

func (a ItemList) Less(i, j int) bool {
	o1 := a[i].(map[string]interface{})
	o2 := a[j].(map[string]interface{})
	return getTime(o1).Unix() > getTime(o2).Unix()
}
func (a ItemList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type K8sListObj struct {
	Kind       string      `json:"kind"`
	ApiVersion string      `json:"apiVersion"`
	Metadata   interface{} `json:"metadata"`
	Items      ItemList    `json:"items"`
}

func (k K8sListObj) Sort() {
	sort.Sort(k.Items)
}

type pageItem struct {
	Metadata interface{} `json:"metadata"`
	Spec     interface{} `json:"spec"`
	Status   interface{} `json:"status"`
}
type metadata struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	CreationTimestamp string `json:"creationTimestamp"`
}

type fieldMatcher interface {
	Match(item interface{}) bool
}

type keywordsMatcher struct {
	keywords string
}

func compatibleClusterVersion(minor int, path *string) {
	p := *path
	if minor <= 18 {
		if strings.Contains(p, "networking.k8s.io/v1") && strings.Contains(p, "ingresses") {
			p = strings.Replace(p, "networking.k8s.io/v1", "networking.k8s.io/v1beta1", -1)
		}
	}
	*path = p
}

func hasNamespaceFilter(path string) bool {
	ss := strings.Split(path, "/")
	for i := range ss {
		if ss[i] == "namespaces" {
			return true
		}
	}
	return false
}

func (n keywordsMatcher) Match(item interface{}) bool {
	pageItem := item.(map[string]interface{})
	if pageItem["metadata"].(map[string]interface{})["namespace"] != nil && pageItem["metadata"].(map[string]interface{})["namespace"].(string) == n.keywords {
		return true
	}
	if strings.Contains(pageItem["metadata"].(map[string]interface{})["name"].(string), n.keywords) {
		return true
	}
	if pageItem["message"] != nil && strings.Contains(strings.ToLower(pageItem["message"].(string)), strings.ToLower(n.keywords)) {
		return true
	}
	return false
}

func withNamespaceAndNameMatcher(keywords string) fieldMatcher {
	return &keywordsMatcher{
		keywords: keywords,
	}
}

func fieldFilter(data []interface{}, fms ...fieldMatcher) []interface{} {
	var result []interface{}
	for i := range data {
		for j := range fms {
			if fms[j].Match(data[i]) {
				result = append(result, data[i])
				break
			}
		}
	}
	return result
}

func pageFilter(num, size int, data []interface{}) (int, []interface{}, error) {
	total := len(data)
	result := make([]interface{}, 0)
	if num*size < len(data) {
		result = data[(num-1)*size : (num * size)]
	} else {
		result = data[(num-1)*size:]
	}
	return total, result, nil
}

func fetchMultiNamespaceResource(client *http.Client, namespaces []string, apiUrl url.URL) (*NamespaceResourceContainer, error) {
	wg := &sync.WaitGroup{}
	var mergedContainer NamespaceResourceContainer
	var responses []*http.Response
	var es []error
	for i := range namespaces {
		wg.Add(1)
		ns := namespaces[i]
		go func() {
			newUrl := apiUrl
			newUrl.Path = addUrlNamespace(apiUrl.Path, ns)
			resp, err := client.Get(newUrl.String())
			if err != nil {
				es = append(es, err)
				wg.Done()
				return
			}
			responses = append(responses, resp)
			wg.Done()
		}()

	}
	wg.Wait()
	var forbidden int
	var forbiddenMessage []string
	for i := range responses {
		r := responses[i]
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if r.StatusCode != http.StatusOK {
			if r.StatusCode == http.StatusForbidden {
				forbidden++
				forbiddenMessage = append(forbiddenMessage, string(body))
				continue
			} else {
				return nil, errors.New(string(body))
			}
		}
		var nc NamespaceResourceContainer
		if err := json.Unmarshal(body, &nc); err != nil {
			return nil, err
		}
		mergedContainer.TypeMeta = nc.TypeMeta
		mergedContainer.ListMeta = nc.ListMeta
		mergedContainer.Items = append(mergedContainer.Items, nc.Items...)
	}
	if len(namespaces) == 1 && forbidden == 1 {
		return nil, errors.New(strings.Join(forbiddenMessage, ""))
	}
	return &mergedContainer, nil
}

func (h *Handler) generateTLSTransport(c *v1Cluster.Cluster, profile session.UserProfile) (http.RoundTripper, error) {
	if profile.IsAdministrator {
		c := kubernetes.NewKubernetes(c)
		adminConfig, err := c.Config()
		if err != nil {
			return nil, err
		}
		return rest.TransportFor(adminConfig)

	}

	binding, err := h.clusterBindingService.GetBindingByClusterNameAndUserName(c.Name, profile.Name, common.DBOptions{})
	if err != nil {
		return nil, err
	}
	kubeConf := &rest.Config{
		Host: c.Spec.Connect.Forward.ApiServer,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
			CertData: binding.Certificate,
			KeyData:  pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: c.PrivateKey}),
		},
	}
	return rest.TransportFor(kubeConf)
}

func ensureProxyPathValid(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func parseResourceName(path string) (string, error) {
	ss := strings.Split(path, "/")
	if len(ss) > 0 {
		return ss[len(ss)-1], nil
	}
	return "", fmt.Errorf("cant not get resource name from url %s", path)
}

func addUrlNamespace(path string, ns string) string {
	ss := strings.Split(path, "/")
	resourceName := ss[len(ss)-1]
	per := ss[:len(ss)-1]
	namespacedSs := append(per, "namespaces", ns, resourceName)
	return strings.Join(namespacedSs, "/")
}

func Install(parent iris.Party) {
	handler := NewHandler()
	sp := parent.Party("/proxy")
	sp.Any("/:name/k8s/{p:path}", handler.KubernetesAPIProxy())
}
