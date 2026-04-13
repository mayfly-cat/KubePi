package audit

import (
	"encoding/json"
	"fmt"
	"strings"
)

type metadataHelper struct {
	Name string `json:"name"`
}

type logHelper struct {
	Name     string         `json:"name"`
	Metadata metadataHelper `json:"metadata"`
}

// WriteLogDraft 与 OperationLog / AuditLog 共用的业务字段（不含 HTTP 审计元数据）。
type WriteLogDraft struct {
	Operation           string
	OperationDomain     string
	SpecificInformation string
}

// BuildWriteLogDraft 根据路由与请求体解析写操作日志草稿；若不应记录则 ok 为 false。
func BuildWriteLogDraft(method, path, currentPath string, body []byte, skipBodyName bool) (draft WriteLogDraft, ok bool) {
	method = strings.ToLower(method)
	if method != "post" && method != "delete" && method != "put" && method != "patch" {
		return WriteLogDraft{}, false
	}

	draft.Operation = InferWriteOperation(method, path, body)

	if strings.Contains(path, "ldap") {
		if strings.Contains(path, "import") {
			draft.Operation = "import"
		}
		if strings.Contains(path, "sync") {
			draft.Operation = "sync"
		}
		if strings.Contains(path, "connect") {
			draft.Operation = "testConnect"
		}
		if strings.Contains(path, "login") {
			draft.Operation = "testLogin"
		}
	}

	pathResource := strings.Split(path, "/")
	if strings.HasPrefix(currentPath, "clusters/:name") {
		if len(pathResource) < 3 {
			draft.OperationDomain = pathResource[0]
			if method != "post" {
				draft.SpecificInformation = pathResource[1]
			}
		} else {
			resourceTypeIndex := 2
			if len(pathResource) > 4 && pathResource[2] == "namespaces" {
				resourceTypeIndex = 4
			}
			draft.OperationDomain = fmt.Sprintf("%s_%s", pathResource[0], pathResource[resourceTypeIndex])
			if method != "post" {
				if len(pathResource) > 3 {
					draft.SpecificInformation = fmt.Sprintf("[%s] %s", pathResource[1], pathResource[3])
				} else {
					draft.SpecificInformation = fmt.Sprintf("[%s] %s", pathResource[1], "-")
				}
			}
		}
	} else {
		parts := strings.Split(currentPath, "/")
		if len(parts) > 0 {
			draft.OperationDomain = parts[0]
		}
		if method != "post" {
			if len(pathResource) > 1 {
				draft.SpecificInformation = pathResource[1]
			} else {
				draft.SpecificInformation = "-"
			}
		}
	}

	if !skipBodyName {
		if method == "post" {
			var req logHelper
			if err := json.Unmarshal(body, &req); err != nil {
				return draft, true
			}
			if len(req.Name) == 0 {
				req.Name = req.Metadata.Name
			}
			if strings.HasPrefix(currentPath, "clusters/:name") {
				if len(pathResource) > 1 {
					draft.SpecificInformation = fmt.Sprintf("[%s] %s", pathResource[1], req.Name)
				}
			} else {
				draft.SpecificInformation = req.Name
			}
		}
	}

	return draft, true
}

// InferWriteOperation 将 HTTP 写方法映射为用户可读的业务动作。
func InferWriteOperation(method, path string, body []byte) string {
	method = strings.ToLower(method)
	if method != "post" && method != "delete" && method != "put" && method != "patch" {
		return method
	}
	if method == "patch" {
		// 面向用户的动作语义：大多数 patch 等价于“修改”
		if workloadOp := detectWorkloadPatchOperation(path, body); workloadOp != "" {
			return workloadOp
		}
		return "put"
	}
	return method
}

func detectWorkloadPatchOperation(path string, body []byte) string {
	lowerPath := strings.ToLower(path)
	if !isWorkloadPath(lowerPath) {
		return ""
	}
	lowerBody := strings.ToLower(string(body))

	switch {
	// 仅在实际回滚 API 或请求体包含 rollbackTo 时识别为回滚。
	// 注意：deployment.kubernetes.io/revision 会出现在普通更新/伸缩/换镜像的请求或对象中，不能用作回滚依据。
	case strings.Contains(lowerPath, "/rollback") ||
		strings.Contains(lowerBody, "rollbackto"):
		return "rollback"
	case strings.Contains(lowerPath, "/reschedule") ||
		strings.Contains(lowerBody, "\"reschedule\"") ||
		strings.Contains(lowerBody, "kubepi.io/rescheduledat"):
		return "reschedule"
	case strings.Contains(lowerBody, "kubectl.kubernetes.io/restartedat"):
		return "restart"
	case strings.Contains(lowerBody, "\"paused\":true"):
		return "pause"
	case strings.Contains(lowerBody, "\"paused\":false"):
		return "resume"
	case strings.Contains(lowerPath, "/scale") || strings.Contains(lowerBody, "\"replicas\""):
		return "scale"
	default:
		return ""
	}
}

func isWorkloadPath(path string) bool {
	return strings.Contains(path, "/deployments/") ||
		strings.Contains(path, "/statefulsets/") ||
		strings.Contains(path, "/daemonsets/") ||
		strings.Contains(path, "/jobs/") ||
		strings.Contains(path, "/cronjobs/")
}
