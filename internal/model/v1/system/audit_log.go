package system

import v1 "github.com/KubeOperator/kubepi/internal/model/v1"

// AuditLog 记录平台 API 写操作审计（在请求处理完成后落库，含 HTTP 状态与客户端信息）。
type AuditLog struct {
	v1.BaseModel        `storm:"inline"`
	v1.Metadata         `storm:"inline"`
	Operator            string `json:"operator"`
	HttpMethod          string `json:"httpMethod"`
	RequestPath         string `json:"requestPath"`
	Resource            string `json:"resource"`
	Operation           string `json:"operation"`
	OperationDomain     string `json:"operationDomain"`
	SpecificInformation string `json:"specificInformation"`
	ClientIp            string `json:"clientIp"`
	UserAgent           string `json:"userAgent"`
	StatusCode          int    `json:"statusCode"`
	Success             bool   `json:"success"`
}
