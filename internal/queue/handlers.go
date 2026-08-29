package queue

import (
	"context"
	"encoding/json"

	"cboard-go/internal/services/email"
	"cboard-go/internal/utils"
)

// EmailSendPayload 邮件发送任务负载
type EmailSendPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SubscriptionLogPayload 订阅日志任务负载
type SubscriptionLogPayload struct {
	SubID          uint                   `json:"sub_id"`
	UserID         uint                   `json:"user_id"`
	ActionType     string                 `json:"action_type"`
	ActionBy       string                 `json:"action_by"`
	ActionByUserID *uint                  `json:"action_by_user_id,omitempty"`
	ClientIP       string                 `json:"client_ip"`
	BeforeData     map[string]interface{} `json:"before_data,omitempty"`
	AfterData      map[string]interface{} `json:"after_data,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
}

// RegisterHandlers 返回任务类型到处理器的映射，供 StartWorker 使用。
func RegisterHandlers() map[string]Handler {
	return map[string]Handler{
		TypeEmailSend:       handleEmailSend,
		TypeSubscriptionLog: handleSubscriptionLog,
	}
}

func handleEmailSend(ctx context.Context, payload []byte) error {
	var p EmailSendPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	return email.NewEmailService().SendEmail(p.To, p.Subject, p.Body)
}

func handleSubscriptionLog(ctx context.Context, payload []byte) error {
	var p SubscriptionLogPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	return utils.CreateSubscriptionLog(p.SubID, p.UserID, p.ActionType, p.ActionBy, p.ActionByUserID, p.ClientIP, p.BeforeData, p.AfterData, p.Reason)
}
