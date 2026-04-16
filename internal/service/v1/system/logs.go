package system

import (
	"encoding/json"
	"fmt"
	"time"

	v1System "github.com/KubeOperator/kubepi/internal/model/v1/system"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	costomStorm "github.com/KubeOperator/kubepi/pkg/storm"
	"github.com/KubeOperator/kubepi/pkg/util/lang"
	"github.com/asdine/storm/v3/q"
	"github.com/google/uuid"
)

type Service interface {
	common.DBService
	CreateOperationLog(log *v1System.OperationLog, options common.DBOptions)
	CreateAuditLog(log *v1System.AuditLog, options common.DBOptions)
	CreateLoginLog(log *v1System.LoginLog, options common.DBOptions)
	SearchOperationLogs(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1System.OperationLog, int, error)
	SearchAuditLogs(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1System.AuditLog, int, error)
	SearchLoginLogs(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1System.LoginLog, int, error)
}

func NewService() Service {
	return &service{}
}

type service struct {
	common.DefaultDBService
}

// emitAuditDebugLog 直接输出 AuditLog JSON，字段与 DB 模型一致，便于日志采集解析。
func emitAuditDebugLog(logItem *v1System.AuditLog) {
	raw, marshalErr := json.Marshal(logItem)
	if marshalErr != nil {
		fmt.Printf("[audit-debug] marshal_failed err=%s\n", marshalErr.Error())
		return
	}
	fmt.Printf("[audit-debug] %s\n", string(raw))
}

func (u *service) CreateOperationLog(log *v1System.OperationLog, options common.DBOptions) {
	db := u.GetDB(options)
	log.UUID = uuid.New().String()
	log.CreateAt = time.Now()
	log.UpdateAt = time.Now()
	if err := db.Save(log); err != nil {
		fmt.Printf("operation log %s by user %s write failure, error is %s", log.Operation, log.Operator, err.Error())
	}
}

func (u *service) CreateAuditLog(log *v1System.AuditLog, options common.DBOptions) {
	db := u.GetDB(options)
	log.UUID = uuid.New().String()
	log.CreateAt = time.Now()
	log.UpdateAt = time.Now()
	if err := db.Save(log); err != nil {
		emitAuditDebugLog(log)
		fmt.Printf("audit log %s %s by user %s write failure, error is %s", log.HttpMethod, log.RequestPath, log.Operator, err.Error())
		return
	}
	emitAuditDebugLog(log)
}

func (u *service) CreateLoginLog(log *v1System.LoginLog, options common.DBOptions) {
	db := u.GetDB(options)
	log.UUID = uuid.New().String()
	log.CreateAt = time.Now()
	log.UpdateAt = time.Now()
	if err := db.Save(log); err != nil {
		fmt.Printf("login logs by user %s write failure, error is %s", log.UserName, err.Error())
	}
}

func (s *service) SearchOperationLogs(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1System.OperationLog, int, error) {
	db := s.GetDB(options)

	var ms []q.Matcher
	for k := range conditions {
		if conditions[k].Field == "quick" {
			ms = append(ms, q.Or(
				costomStorm.Like("Operator", conditions[k].Value),
				costomStorm.Like("Operation", conditions[k].Value),
				costomStorm.Like("SpecificInformation", conditions[k].Value),
			))
		} else {
			field := lang.FirstToUpper(conditions[k].Field)
			value := conditions[k].Value
			cmpValue := interface{}(value)
			if field == "Success" {
				cmpValue = lang.ParseValueType(value)
			}

			switch conditions[k].Operator {
			case "eq":
				ms = append(ms, q.Eq(field, cmpValue))
			case "ne":
				ms = append(ms, q.Not(q.Eq(field, cmpValue)))
			case "like":
				ms = append(ms, costomStorm.Like(field, value))
			case "not like":
				ms = append(ms, q.Not(costomStorm.Like(field, value)))
			}
		}
	}
	query := db.Select(ms...).OrderBy("CreateAt").Reverse()
	count, err := query.Count(&v1System.OperationLog{})
	if err != nil {
		return nil, 0, err
	}
	if size != 0 {
		query.Limit(size).Skip((num - 1) * size)
	}
	logs := make([]v1System.OperationLog, 0)
	if err := query.Find(&logs); err != nil {
		return nil, 0, err
	}
	return logs, count, nil
}

func (s *service) SearchAuditLogs(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1System.AuditLog, int, error) {
	db := s.GetDB(options)

	var ms []q.Matcher
	for k := range conditions {
		if conditions[k].Field == "quick" {
			ms = append(ms, q.Or(
				costomStorm.Like("Cluster", conditions[k].Value),
				costomStorm.Like("Operator", conditions[k].Value),
				costomStorm.Like("Operation", conditions[k].Value),
				costomStorm.Like("OperationDomain", conditions[k].Value),
				costomStorm.Like("SpecificInformation", conditions[k].Value),
				costomStorm.Like("OperationRecord", conditions[k].Value),
				costomStorm.Like("ClientIp", conditions[k].Value),
			))
		} else {
			field := lang.FirstToUpper(conditions[k].Field)
			value := conditions[k].Value
			cmpValue := interface{}(value)
			if field == "Success" {
				cmpValue = lang.ParseValueType(value)
			}

			switch conditions[k].Operator {
			case "eq":
				ms = append(ms, q.Eq(field, cmpValue))
			case "ne":
				ms = append(ms, q.Not(q.Eq(field, cmpValue)))
			case "like":
				ms = append(ms, costomStorm.Like(field, value))
			case "not like":
				ms = append(ms, q.Not(costomStorm.Like(field, value)))
			}
		}
	}
	query := db.Select(ms...).OrderBy("CreateAt").Reverse()
	count, err := query.Count(&v1System.AuditLog{})
	if err != nil {
		return nil, 0, err
	}
	if size != 0 {
		query.Limit(size).Skip((num - 1) * size)
	}
	logs := make([]v1System.AuditLog, 0)
	if err := query.Find(&logs); err != nil {
		return nil, 0, err
	}
	return logs, count, nil
}

func (s *service) SearchLoginLogs(num, size int, conditions common.Conditions, options common.DBOptions) ([]v1System.LoginLog, int, error) {
	db := s.GetDB(options)

	var ms []q.Matcher
	for k := range conditions {
		if conditions[k].Field == "quick" {
			ms = append(ms, q.Or(
				costomStorm.Like("UserName", conditions[k].Value),
				costomStorm.Like("Ip", conditions[k].Value),
				costomStorm.Like("City", conditions[k].Value),
			))
		} else {
			field := lang.FirstToUpper(conditions[k].Field)
			value := conditions[k].Value

			switch conditions[k].Operator {
			case "eq":
				ms = append(ms, q.Eq(field, value))
			case "ne":
				ms = append(ms, q.Not(q.Eq(field, value)))
			case "like":
				ms = append(ms, costomStorm.Like(field, value))
			case "not like":
				ms = append(ms, q.Not(costomStorm.Like(field, value)))
			}
		}
	}
	query := db.Select(ms...).OrderBy("CreateAt").Reverse()
	count, err := query.Count(&v1System.LoginLog{})
	if err != nil {
		return nil, 0, err
	}
	if size != 0 {
		query.Limit(size).Skip((num - 1) * size)
	}
	logs := make([]v1System.LoginLog, 0)
	if err := query.Find(&logs); err != nil {
		return nil, 0, err
	}
	return logs, count, nil
}
