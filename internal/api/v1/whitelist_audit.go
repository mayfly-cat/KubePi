package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KubeOperator/kubepi/internal/api/v1/session"
	fileModel "github.com/KubeOperator/kubepi/internal/model/v1/file"
	v1System "github.com/KubeOperator/kubepi/internal/model/v1/system"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	v1SystemService "github.com/KubeOperator/kubepi/internal/service/v1/system"
	"github.com/KubeOperator/kubepi/pkg/util/requestip"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

// whitelistAuditHandler 为 resourceWhiteList 中的资源补写审计（原 logHandler 会跳过白名单）。
// proxy / ws 仍由各自逻辑处理或不记在此（proxy 写操作已有专用审计）。
func whitelistAuditHandler() iris.Handler {
	return func(ctx *context.Context) {
		resource := ctx.Values().GetString("resource")
		if resource == "" || !resourceWhiteList.In(resource) {
			ctx.Next()
			return
		}
		if resource == "proxy" || resource == "ws" {
			ctx.Next()
			return
		}

		method := strings.ToLower(ctx.Method())
		path := ctx.Request().URL.Path
		if !shouldAuditWhitelistedRoute(resource, method, path) {
			ctx.Next()
			return
		}

		var body []byte
		if method == "post" || method == "put" || method == "patch" {
			data, err := ctx.GetBody()
			if err == nil {
				body = data
				ctx.Request().Body = nopCloserBytes(data)
			}
		}

		ctx.Next()

		p := ctx.Values().Get("profile")
		profile, ok := p.(session.UserProfile)
		if !ok || profile.Name == "" {
			return
		}

		status := ctx.GetStatusCode()
		success := status >= 200 && status < 400
		domain, record, cluster := buildWhitelistAuditDetail(ctx, resource, method, path, body)

		auditLog := v1System.AuditLog{
			Operator:            profile.Name,
			Cluster:             cluster,
			HttpMethod:          method,
			RequestPath:         path,
			Resource:            resource,
			Operation:           method,
			OperationDomain:     domain,
			SpecificInformation: whitelistSpecificHint(path),
			OperationRecord:     record,
			ClientIp:            requestip.FromRequest(ctx.Request()),
			UserAgent:           ctx.GetHeader("User-Agent"),
			StatusCode:          status,
			Success:             success,
		}
		go v1SystemService.NewService().CreateAuditLog(&auditLog, common.DBOptions{})
	}
}

func shouldAuditWhitelistedRoute(resource, method, path string) bool {
	if method == "get" {
		return resource == "pod" && strings.Contains(path, "/pod/files/download/")
	}
	if method != "post" && method != "put" && method != "patch" && method != "delete" {
		return false
	}
	switch resource {
	case "sessions":
		// 登录单独有 LoginLog，此处不重复审计
		return method != "post"
	case "pod", "charts", "apps", "mfa", "webkubectl":
		return true
	default:
		return false
	}
}

func whitelistSpecificHint(path string) string {
	if idx := strings.Index(path, "/v1/"); idx >= 0 {
		return path[idx+4:]
	}
	return path
}

func segmentAfterMarker(path, marker string) string {
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	rest := strings.Trim(path[idx+len(marker):], "/")
	seg, _, _ := strings.Cut(rest, "/")
	return seg
}

func buildWhitelistAuditDetail(ctx *context.Context, resource, method, path string, body []byte) (domain, record, cluster string) {
	switch resource {
	case "pod":
		domain = "pod_files"
		var fr fileModel.Request
		_ = json.Unmarshal(body, &fr)
		if fr.Cluster != "" {
			cluster = fr.Cluster
		}
		if method == "get" && strings.Contains(path, "/files/download/") {
			cluster = ctx.URLParam("cluster")
			record = fmt.Sprintf("数据导出(%s)：cluster=%s namespace=%s pod=%s container=%s path=%s",
				pathSuffix(path), ctx.URLParam("cluster"), ctx.URLParam("namespace"),
				ctx.URLParam("podName"), ctx.URLParam("containerName"), ctx.URLParam("path"))
			return domain, record, cluster
		}
		action := podFileAction(path)
		record = fmt.Sprintf("%s：cluster=%s namespace=%s pod=%s container=%s path=%s",
			action, fr.Cluster, fr.Namespace, fr.PodName, fr.ContainerName, fr.Path)
		if fr.OldPath != "" {
			record += fmt.Sprintf(" oldPath=%s", fr.OldPath)
		}
		return domain, record, cluster

	case "charts":
		domain = "charts"
		cluster = segmentAfterMarker(path, "/charts/")
		record = fmt.Sprintf("Chart 仓库/应用操作：%s %s", strings.ToUpper(method), whitelistSpecificHint(path))
		return domain, record, cluster

	case "apps":
		domain = "chart_apps"
		cluster = segmentAfterMarker(path, "/apps/")
		record = fmt.Sprintf("已安装应用操作：%s %s", strings.ToUpper(method), whitelistSpecificHint(path))
		return domain, record, cluster

	case "mfa":
		domain = "mfa"
		if strings.Contains(path, "/mfa/bind") {
			record = "MFA：绑定/更新验证器"
		} else if strings.Contains(path, "/mfa/valid") {
			record = "MFA：校验动态口令"
		} else {
			record = fmt.Sprintf("MFA 操作：%s", whitelistSpecificHint(path))
		}
		return domain, record, cluster

	case "webkubectl":
		domain = "webkubectl"
		var w struct {
			Cluster string `json:"cluster"`
		}
		_ = json.Unmarshal(body, &w)
		cluster = w.Cluster
		record = fmt.Sprintf("Webkubectl：创建终端会话 cluster=%s", w.Cluster)
		return domain, record, cluster

	case "sessions":
		domain = "sessions"
		switch method {
		case "delete":
			record = "会话：退出登录"
		case "put":
			if strings.Contains(path, "/sessions/password") {
				record = "账户：修改密码"
			} else {
				record = "账户：更新个人资料"
			}
		default:
			record = fmt.Sprintf("会话/账户：%s %s", strings.ToUpper(method), whitelistSpecificHint(path))
		}
		return domain, record, cluster

	default:
		return resource, fmt.Sprintf("%s %s", strings.ToUpper(method), path), cluster
	}
}

func pathSuffix(path string) string {
	if strings.Contains(path, "/files/download/file") {
		return "file"
	}
	if strings.Contains(path, "/files/download/folder") {
		return "folder"
	}
	return "download"
}

func podFileAction(path string) string {
	switch {
	case strings.Contains(path, "/folder/create"):
		return "容器文件：创建目录"
	case strings.Contains(path, "/folder/delete"):
		return "容器文件：删除目录"
	case strings.Contains(path, "/files/create"):
		return "容器文件：新建文件"
	case strings.Contains(path, "/files/open"):
		return "容器文件：读取文件"
	case strings.Contains(path, "/files/rename"):
		return "容器文件：重命名"
	case strings.Contains(path, "/files/upload"):
		return "容器文件：上传"
	case strings.Contains(path, "/files/update"):
		return "容器文件：更新文件"
	case strings.HasSuffix(path, "/pod/files"):
		return "容器文件：列出目录/文件"
	default:
		return "容器文件：操作"
	}
}
