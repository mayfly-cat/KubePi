package ingresshistory

import (
	"encoding/json"
	"fmt"

	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	ingressHistoryService "github.com/KubeOperator/kubepi/internal/service/v1/ingresshistory"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

type Handler struct {
	ingressHistoryService ingressHistoryService.Service
}

func NewHandler() *Handler {
	return &Handler{
		ingressHistoryService: ingressHistoryService.NewService(),
	}
}

// ListHistory 获取 Ingress 历史版本列表
// @Tags ingresshistory
// @Summary List ingress history
// @Description List ingress history versions
// @Accept json
// @Produce json
// @Param clusterName path string true "Cluster name"
// @Param namespace path string true "Namespace"
// @Param ingressName path string true "Ingress name"
// @Success 200 {object} []map[string]interface{}
// @Security ApiKeyAuth
// @Router /clusters/{clusterName}/namespaces/{namespace}/ingresses/{ingressName}/history [get]
func (h *Handler) ListHistory() iris.Handler {
	return func(ctx *context.Context) {
		clusterName := ctx.Params().GetString("clusterName")
		namespace := ctx.Params().GetString("namespace")
		ingressName := ctx.Params().GetString("ingressName")

		if clusterName == "" {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", "clusterName is required")
			return
		}
		if namespace == "" {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", "namespace is required")
			return
		}
		if ingressName == "" {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", "ingressName is required")
			return
		}

		histories, err := h.ingressHistoryService.ListHistory(clusterName, namespace, ingressName, common.DBOptions{})
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", fmt.Sprintf("list history failed: %s", err.Error()))
			return
		}

		ctx.Values().Set("data", histories)
	}
}

// GetHistory 获取指定版本的历史记录
// @Tags ingresshistory
// @Summary Get ingress history version
// @Description Get specific version of ingress history
// @Accept json
// @Produce json
// @Param clusterName path string true "Cluster name"
// @Param namespace path string true "Namespace"
// @Param ingressName path string true "Ingress name"
// @Param version path int true "Version number"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /clusters/{clusterName}/namespaces/{namespace}/ingresses/{ingressName}/history/{version} [get]
func (h *Handler) GetHistory() iris.Handler {
	return func(ctx *context.Context) {
		clusterName := ctx.Params().GetString("clusterName")
		namespace := ctx.Params().GetString("namespace")
		ingressName := ctx.Params().GetString("ingressName")
		version, err := ctx.Params().GetInt("version")
		if err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", "invalid version number")
			return
		}

		history, err := h.ingressHistoryService.GetHistory(clusterName, namespace, ingressName, version, common.DBOptions{})
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", fmt.Sprintf("get history failed: %s", err.Error()))
			return
		}

		// 解析 IngressData 为 JSON 对象
		var ingressData interface{}
		if err := json.Unmarshal(history.IngressData, &ingressData); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", fmt.Sprintf("parse ingress data failed: %s", err.Error()))
			return
		}

		result := map[string]interface{}{
			"history":     history,
			"ingressData": ingressData,
		}

		ctx.Values().Set("data", result)
	}
}

// Rollback 回滚到指定版本
// @Tags ingresshistory
// @Summary Rollback ingress to version
// @Description Rollback ingress to specific version
// @Accept json
// @Produce json
// @Param clusterName path string true "Cluster name"
// @Param namespace path string true "Namespace"
// @Param ingressName path string true "Ingress name"
// @Param version path int true "Version number to rollback"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /clusters/{clusterName}/namespaces/{namespace}/ingresses/{ingressName}/history/{version}/rollback [post]
func (h *Handler) Rollback() iris.Handler {
	return func(ctx *context.Context) {
		clusterName := ctx.Params().GetString("clusterName")
		namespace := ctx.Params().GetString("namespace")
		ingressName := ctx.Params().GetString("ingressName")
		version, err := ctx.Params().GetInt("version")
		if err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Values().Set("message", "invalid version number")
			return
		}

		// 获取历史版本
		history, err := h.ingressHistoryService.GetHistory(clusterName, namespace, ingressName, version, common.DBOptions{})
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", fmt.Sprintf("get history failed: %s", err.Error()))
			return
		}

		// 解析 IngressData
		var ingressData interface{}
		if err := json.Unmarshal(history.IngressData, &ingressData); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Values().Set("message", fmt.Sprintf("parse ingress data failed: %s", err.Error()))
			return
		}

		// 直接返回 Ingress 数据，前端可以通过 proxy API 更新
		// 更新时会自动保存当前版本为历史
		ctx.Values().Set("data", ingressData)
	}
}

func Install(parent iris.Party) {
	handler := NewHandler()
	sp := parent.Party("/clusters/:clusterName/namespaces/:namespace/ingresses/:ingressName/history")
	sp.Get("", handler.ListHistory())
	sp.Get("/:version", handler.GetHistory())
	sp.Post("/:version/rollback", handler.Rollback())
}
