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
	if method != "post" && method != "delete" && method != "put" {
		return WriteLogDraft{}, false
	}

	draft.Operation = method

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
			draft.OperationDomain = fmt.Sprintf("%s_%s", pathResource[0], pathResource[2])
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
