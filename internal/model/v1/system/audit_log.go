package system

import v1 "github.com/KubeOperator/kubepi/internal/model/v1"

// AuditLog 记录平台 API 写操作审计（在请求处理完成后落库，含 HTTP 状态与客户端信息）。
type AuditLog struct {
	v1.BaseModel        `storm:"inline"`
	v1.Metadata         `storm:"inline"`
	Operator            string `json:"operator"`
	Cluster             string `json:"cluster"`
	HttpMethod          string `json:"httpMethod"`
	RequestPath         string `json:"requestPath"`
	Resource            string `json:"resource"`
	Operation           string `json:"operation"`
	OperationDomain     string `json:"operationDomain"`
	SpecificInformation string `json:"specificInformation"`
	// OperationRecord 人类可读的操作说明（如 Ingress 规则增删、文件导出参数等）
	OperationRecord string   `json:"operationRecord"`
	RuleAdds        []string `json:"ruleAdds"`
	RuleRemoves     []string `json:"ruleRemoves"`
	ClientIp        string   `json:"clientIp"`
	UserAgent       string   `json:"userAgent"`
	StatusCode      int      `json:"statusCode"`
	Success         bool     `json:"success"`
}
