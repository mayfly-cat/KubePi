package ingresshistory

import (
	v1 "github.com/KubeOperator/kubepi/internal/model/v1"
)

// IngressHistory 存储 Ingress 的历史版本
type IngressHistory struct {
	v1.BaseModel `storm:"inline"`
	v1.Metadata  `storm:"inline"`

	// 关联信息
	ClusterName string `json:"clusterName" storm:"index"`
	Namespace   string `json:"namespace" storm:"index"`
	IngressName string `json:"ingressName" storm:"index"`

	// 版本信息
	Version     int    `json:"version" storm:"index"` // 版本号，从1开始递增
	Description string `json:"description"`           // 版本描述（可选）

	// Ingress 完整配置（JSON格式）
	IngressData []byte `json:"ingressData"`

	// 操作信息
	Operator string `json:"operator"` // 操作人
}

// 复合索引：按集群、命名空间、Ingress名称和版本号查询
// Storm 会自动处理这些索引
