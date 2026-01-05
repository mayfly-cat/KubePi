package ingresshistory

import (
	"encoding/json"
	"fmt"
	"time"

	v1 "github.com/KubeOperator/kubepi/internal/model/v1"
	v1IngressHistory "github.com/KubeOperator/kubepi/internal/model/v1/ingresshistory"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/google/uuid"
)

type Service interface {
	common.DBService
	// SaveHistory 保存 Ingress 历史版本
	SaveHistory(clusterName, namespace, ingressName string, ingressData interface{}, operator, description string, options common.DBOptions) (*v1IngressHistory.IngressHistory, error)
	// ListHistory 获取指定 Ingress 的历史版本列表
	ListHistory(clusterName, namespace, ingressName string, options common.DBOptions) ([]v1IngressHistory.IngressHistory, error)
	// GetHistory 获取指定版本的历史记录
	GetHistory(clusterName, namespace, ingressName string, version int, options common.DBOptions) (*v1IngressHistory.IngressHistory, error)
	// GetLatestVersion 获取最新版本号
	GetLatestVersion(clusterName, namespace, ingressName string, options common.DBOptions) (int, error)
	// DeleteHistory 删除指定版本的历史记录
	DeleteHistory(clusterName, namespace, ingressName string, version int, options common.DBOptions) error
	// DeleteAllHistory 删除指定 Ingress 的所有历史记录
	DeleteAllHistory(clusterName, namespace, ingressName string, options common.DBOptions) error
}

func NewService() Service {
	return &service{}
}

type service struct {
	common.DefaultDBService
}

func (s *service) SaveHistory(clusterName, namespace, ingressName string, ingressData interface{}, operator, description string, options common.DBOptions) (*v1IngressHistory.IngressHistory, error) {
	db := s.GetDB(options)

	// 获取当前最大版本号
	latestVersion, err := s.GetLatestVersion(clusterName, namespace, ingressName, options)
	if err != nil && err != storm.ErrNotFound {
		return nil, fmt.Errorf("get latest version failed: %w", err)
	}

	// 序列化 Ingress 数据
	ingressBytes, err := json.Marshal(ingressData)
	if err != nil {
		return nil, fmt.Errorf("marshal ingress data failed: %w", err)
	}

	// 创建历史记录
	history := &v1IngressHistory.IngressHistory{
		BaseModel: v1.BaseModel{
			ApiVersion: "v1",
			Kind:       "IngressHistory",
			CreateAt:   time.Now(),
			UpdateAt:   time.Now(),
			CreatedBy:  operator,
		},
		Metadata: v1.Metadata{
			Name:        fmt.Sprintf("%s-%s-%s-v%d", clusterName, namespace, ingressName, latestVersion+1),
			Description: description,
			UUID:        uuid.New().String(),
		},
		ClusterName: clusterName,
		Namespace:   namespace,
		IngressName: ingressName,
		Version:     latestVersion + 1,
		Description: description,
		IngressData: ingressBytes,
		Operator:    operator,
	}

	if err := db.Save(history); err != nil {
		return nil, fmt.Errorf("save history failed: %w", err)
	}

	return history, nil
}

func (s *service) ListHistory(clusterName, namespace, ingressName string, options common.DBOptions) ([]v1IngressHistory.IngressHistory, error) {
	db := s.GetDB(options)

	var histories []v1IngressHistory.IngressHistory
	query := db.Select(
		q.Eq("ClusterName", clusterName),
		q.Eq("Namespace", namespace),
		q.Eq("IngressName", ingressName),
	).OrderBy("Version").Reverse()

	if err := query.Find(&histories); err != nil {
		if err == storm.ErrNotFound {
			return []v1IngressHistory.IngressHistory{}, nil
		}
		return nil, fmt.Errorf("list history failed: %w", err)
	}

	return histories, nil
}

func (s *service) GetHistory(clusterName, namespace, ingressName string, version int, options common.DBOptions) (*v1IngressHistory.IngressHistory, error) {
	db := s.GetDB(options)

	var history v1IngressHistory.IngressHistory
	query := db.Select(
		q.Eq("ClusterName", clusterName),
		q.Eq("Namespace", namespace),
		q.Eq("IngressName", ingressName),
		q.Eq("Version", version),
	)

	if err := query.First(&history); err != nil {
		return nil, fmt.Errorf("get history failed: %w", err)
	}

	return &history, nil
}

func (s *service) GetLatestVersion(clusterName, namespace, ingressName string, options common.DBOptions) (int, error) {
	db := s.GetDB(options)

	var histories []v1IngressHistory.IngressHistory
	query := db.Select(
		q.Eq("ClusterName", clusterName),
		q.Eq("Namespace", namespace),
		q.Eq("IngressName", ingressName),
	).OrderBy("Version").Reverse().Limit(1)

	if err := query.Find(&histories); err != nil {
		if err == storm.ErrNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("get latest version failed: %w", err)
	}

	if len(histories) == 0 {
		return 0, nil
	}

	return histories[0].Version, nil
}

func (s *service) DeleteHistory(clusterName, namespace, ingressName string, version int, options common.DBOptions) error {
	db := s.GetDB(options)

	history, err := s.GetHistory(clusterName, namespace, ingressName, version, options)
	if err != nil {
		return err
	}

	if err := db.DeleteStruct(history); err != nil {
		return fmt.Errorf("delete history failed: %w", err)
	}

	return nil
}

func (s *service) DeleteAllHistory(clusterName, namespace, ingressName string, options common.DBOptions) error {
	db := s.GetDB(options)

	histories, err := s.ListHistory(clusterName, namespace, ingressName, options)
	if err != nil {
		return err
	}

	for i := range histories {
		if err := db.DeleteStruct(&histories[i]); err != nil {
			return fmt.Errorf("delete history %d failed: %w", histories[i].Version, err)
		}
	}

	return nil
}
