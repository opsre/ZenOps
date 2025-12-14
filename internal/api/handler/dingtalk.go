package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/dingtalk"
)

// DingTalkHandler 钉钉回调处理器
type DingTalkHandler struct {
	handler *dingtalk.MessageHandler
	crypto  *dingtalk.CallbackCrypto
}

// NewDingTalkHandler 创建钉钉处理器
func NewDingTalkHandler(handler *dingtalk.MessageHandler, crypto *dingtalk.CallbackCrypto) *DingTalkHandler {
	return &DingTalkHandler{
		handler: handler,
		crypto:  crypto,
	}
}

// HandleCallback 处理钉钉回调
func (h *DingTalkHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	logx.Info("Received DingTalk callback, method %s, remote_addr %s",
		r.Method,
		r.RemoteAddr)

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logx.Error("Failed to read request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	logx.Debug("Request body: %v", string(body))
	// 验证签名
	timestamp := r.Header.Get("Timestamp")
	nonce := r.Header.Get("Nonce")
	signature := r.Header.Get("Signature")

	if !dingtalk.IsValidTimestamp(timestamp) {
		logx.Warn("Invalid timestamp: %s", timestamp)
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}

	if !h.crypto.VerifySignature(timestamp, nonce, string(body), signature) {
		logx.Warn("Signature verification failed: timestamp %s, nonce %s, signature %s",
			timestamp,
			nonce,
			signature)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// 解析加密消息
	var callbackReq dingtalk.CallbackRequest
	if err := json.Unmarshal(body, &callbackReq); err != nil {
		logx.Error("Failed to parse callback request: %v", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// 解密消息
	msg, err := h.crypto.DecryptMessage(callbackReq.Encrypt)
	if err != nil {
		logx.Error("Failed to decrypt message: %v", err)
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	logx.Info("Decrypted message: sender %s, msg_type %s, conversation_id %s",
		msg.SenderNick,
		msg.MsgType,
		msg.ConversationID)

	// 处理消息
	resp, err := h.handler.HandleMessage(r.Context(), msg)
	if err != nil {
		logx.Error("Failed to handle message: %v", err)
		http.Error(w, "Message handling failed", http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logx.Error("Failed to encode response: %v", err)
		return
	}

	logx.Info("Callback handled successfully")
}

// HandleWebhook 处理 Webhook(用于主动发送消息测试)
func (h *DingTalkHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversation_id"`
		Message        string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 这里可以添加主动发送消息的逻辑
	// 暂时返回成功
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Webhook received",
	})
}

// HandleHealthCheck 健康检查
func (h *DingTalkHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "dingtalk",
	})
}
