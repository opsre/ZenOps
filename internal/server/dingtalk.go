package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

// ==================== 钉钉加解密 ====================

// DingTalkCrypto 回调加解密工具
type DingTalkCrypto struct {
	token          string
	encodingAESKey string
	suiteKey       string
	aesKey         []byte
}

// NewDingTalkCrypto 创建回调加解密工具
func NewDingTalkCrypto(token, encodingAESKey, suiteKey string) (*DingTalkCrypto, error) {
	if len(encodingAESKey) != 43 {
		return nil, fmt.Errorf("invalid encoding aes key length: %d", len(encodingAESKey))
	}

	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("failed to decode aes key: %w", err)
	}

	return &DingTalkCrypto{
		token:          token,
		encodingAESKey: encodingAESKey,
		suiteKey:       suiteKey,
		aesKey:         aesKey,
	}, nil
}

// VerifySignature 验证签名
func (c *DingTalkCrypto) VerifySignature(timestamp, nonce, body, signature string) bool {
	message := timestamp + "\n" + nonce + "\n" + body
	mac := hmac.New(sha256.New, []byte(c.token))
	mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// DecryptMessage 解密消息
func (c *DingTalkCrypto) DecryptMessage(encryptedMsg string) (*DingTalkMessage, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		return nil, err
	}

	block, _ := aes.NewCipher(c.aesKey)
	iv := c.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)
	plaintext = pkcs7UnPadding(plaintext)

	if len(plaintext) < 20 {
		return nil, fmt.Errorf("plaintext too short")
	}

	msgLen := int(plaintext[16])<<24 | int(plaintext[17])<<16 | int(plaintext[18])<<8 | int(plaintext[19])
	msgContent := plaintext[20 : 20+msgLen]

	var msg DingTalkMessage
	_ = json.Unmarshal(msgContent, &msg)
	return &msg, nil
}

func pkcs7UnPadding(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	unpadding := int(data[length-1])
	if unpadding > length {
		return data
	}
	return data[:(length - unpadding)]
}

// ==================== 数据结构 ====================

// DingTalkMessage 钉钉消息
type DingTalkMessage struct {
	MsgID            string           `json:"msgId"`
	MsgType          string           `json:"msgtype"`
	CreateAt         int64            `json:"createAt"`
	ConversationID   string           `json:"conversationId"`
	ConversationType string           `json:"conversationType"` // "1"=单聊, "2"=群聊
	SenderID         string           `json:"senderId"`
	SenderStaffID    string           `json:"senderStaffId"`
	SenderNick       string           `json:"senderNick"`
	ChatbotUserID    string           `json:"chatbotUserId"`
	Text             *DingTalkText    `json:"text,omitempty"`
	AtUsers          []DingTalkAtUser `json:"atUsers,omitempty"`
}

type DingTalkText struct {
	Content string `json:"content"`
}

type DingTalkAtUser struct {
	DingtalkID string `json:"dingtalkId"`
}

type DingTalkResponse struct {
	MsgType  string               `json:"msgtype"`
	Text     *DingTalkTextMsg     `json:"text,omitempty"`
	Markdown *DingTalkMarkdownMsg `json:"markdown,omitempty"`
}

type DingTalkTextMsg struct {
	Content string `json:"content"`
}

type DingTalkMarkdownMsg struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// ==================== 意图解析 ====================

// DingTalkIntent 用户意图
type DingTalkIntent struct {
	MCPTool string
	Params  map[string]any
}

// ParseIntent 解析用户意图
func ParseIntent(message string) (*DingTalkIntent, error) {
	// 简化的意图识别
	patterns := []struct {
		regex   *regexp.Regexp
		tool    string
		extract func([]string) map[string]any
	}{
		// 阿里云 ECS - IP 搜索
		{regexp.MustCompile(`(?i)(IP|ip).*([\d\.]+)`), "search_ecs_by_ip", func(m []string) map[string]any {
			return map[string]any{"ip": m[2]}
		}},
		// 阿里云 ECS - 名称搜索
		{regexp.MustCompile(`(?i)(名称?|名字|叫).*([\w\-]+)`), "search_ecs_by_name", func(m []string) map[string]any {
			return map[string]any{"name": m[2]}
		}},
		// 阿里云 ECS - 列表
		{regexp.MustCompile(`(?i)(列出|查询|查看).*(阿里云|ECS|ecs|服务器)`), "list_ecs", func(m []string) map[string]any {
			params := make(map[string]any)
			if strings.Contains(m[0], "杭州") {
				params["region"] = "cn-hangzhou"
			}
			return params
		}},
		// 腾讯云 CVM
		{regexp.MustCompile(`(?i)(腾讯云|CVM|cvm)`), "list_cvm", func(m []string) map[string]any {
			return make(map[string]any)
		}},
		// Jenkins
		{regexp.MustCompile(`(?i)(jenkins|Jenkins|任务)`), "list_jenkins_jobs", func(m []string) map[string]any {
			return make(map[string]any)
		}},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(message); matches != nil {
			return &DingTalkIntent{
				MCPTool: p.tool,
				Params:  p.extract(matches),
			}, nil
		}
	}

	return nil, fmt.Errorf("无法识别您的请求")
}

// IsValidTimestamp 检查时间戳有效性
func IsValidTimestamp(timestamp string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().UnixMilli()
	diff := now - ts
	return diff >= 0 && diff <= 5*60*1000
}

// ExtractUserMessage 提取用户消息
func ExtractUserMessage(msg *DingTalkMessage) string {
	if msg.Text == nil {
		return ""
	}
	content := msg.Text.Content
	// 去除 @机器人
	for _, at := range msg.AtUsers {
		if at.DingtalkID == msg.ChatbotUserID {
			content = strings.ReplaceAll(content, "@"+msg.ChatbotUserID, "")
		}
	}
	return strings.TrimSpace(content)
}

// ==================== 钉钉消息处理 ====================

// DingTalkMessageHandler 消息处理器
type DingTalkMessageHandler struct {
	streamClient   *DingTalkStreamClient
	mcpServer      *imcp.MCPServer
	config         *config.Config
	chatLogService *service.ChatLogService
}

// NewDingTalkMessageHandler 创建消息处理器
func NewDingTalkMessageHandler(cfg *config.Config, mcpServer *imcp.MCPServer) *DingTalkMessageHandler {
	// 创建流式客户端
	streamClient, err := NewDingTalkStreamClient(cfg.DingTalk.AppKey, cfg.DingTalk.AppSecret, cfg.DingTalk.CardTemplateID)
	if err != nil {
		logx.Error("Failed to create stream client: %v", err)
		return nil
	}

	return &DingTalkMessageHandler{
		streamClient:   streamClient,
		mcpServer:      mcpServer,
		config:         cfg,
		chatLogService: service.NewChatLogService(),
	}
}

// HandleMessage 处理消息
func (h *DingTalkMessageHandler) HandleMessage(ctx context.Context, msg *DingTalkMessage) (*DingTalkResponse, error) {
	userMessage := ExtractUserMessage(msg)
	if userMessage == "" {
		return &DingTalkResponse{
			MsgType: "text",
			Text:    &DingTalkTextMsg{Content: "请输入您的查询内容"},
		}, nil
	}

	logx.Info("Processing DingTalk message, sender=%s, message=%s", msg.SenderNick, userMessage)

	// 确定消息来源（私聊/群聊）
	source := "私聊"
	if msg.ConversationType == "2" {
		source = "群聊"
	}

	// 保存用户消息到数据库
	username := msg.SenderNick
	if username == "" {
		username = msg.SenderStaffID
	}
	userLog, err := h.chatLogService.CreateUserMessage(username, source, userMessage)
	if err != nil {
		logx.Error("Failed to save user message to database: %v", err)
	}

	// 帮助命令
	if strings.Contains(userMessage, "帮助") {
		helpText := "发送资源查询请求,如\"查询阿里云 ECS\"、\"列出 Jenkins 任务\"等"

		// 保存帮助消息到数据库
		if userLog != nil {
			_, saveErr := h.chatLogService.CreateAIMessage(username, source, helpText, userLog.ID)
			if saveErr != nil {
				logx.Error("Failed to save help message to database: %v", saveErr)
			}
		}

		return &DingTalkResponse{
			MsgType: "text",
			Text:    &DingTalkTextMsg{Content: helpText},
		}, nil
	}

	// 解析意图
	intent, parseErr := ParseIntent(userMessage)
	if parseErr != nil {
		return &DingTalkResponse{
			MsgType: "text",
			Text:    &DingTalkTextMsg{Content: parseErr.Error()},
		}, nil
	}

	// 异步处理查询
	go h.processQueryAsync(ctx, msg, intent, username, source, userLog)

	return &DingTalkResponse{
		MsgType: "text",
		Text:    &DingTalkTextMsg{Content: "🔍 正在查询,请稍候..."},
	}, nil
}

// processQueryAsync 异步处理查询
func (h *DingTalkMessageHandler) processQueryAsync(ctx context.Context, msg *DingTalkMessage, intent *DingTalkIntent, username, source string, userLog *model.ChatLog) {
	question := ExtractUserMessage(msg)
	trackID := fmt.Sprintf("track_%s_%d", msg.MsgID, time.Now().Unix())

	// 1. 创建并投递 AI 卡片
	if err := h.streamClient.CreateAndDeliverCard(ctx, trackID, msg.ConversationID, msg.ConversationType, msg.SenderStaffID); err != nil {
		logx.Error("Failed to create and deliver card: %v", err)
		return
	}

	// 2. 发送初始提示
	if err := h.streamClient.StreamInitial(trackID, question); err != nil {
		logx.Error("Failed to send initial message: %v", err)
	}

	// 3. 调用 MCP 工具获取结果
	result, err := h.callMCPTool(ctx, intent)
	if err != nil {
		logx.Error("Failed to call MCP tool: %v", err)
		_ = h.streamClient.StreamError(trackID, err, question)
		return
	}

	// 4. 创建内容通道并流式发送
	contentCh := make(chan string, 10)
	go func() {
		// 模拟流式输出:将结果按行分批发送
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if line != "" {
				contentCh <- line + "\n"
				time.Sleep(50 * time.Millisecond) // 模拟打字效果
			}
		}
		close(contentCh)
	}()

	// 5. 流式更新卡片
	h.streamClient.StreamResponse(ctx, trackID, contentCh, question)

	// 6. 保存AI响应到数据库
	if userLog != nil && result != "" {
		_, saveErr := h.chatLogService.CreateAIMessage(username, source, result, userLog.ID)
		if saveErr != nil {
			logx.Error("Failed to save AI response to database: %v", saveErr)
		}
	}
}

// callMCPTool 调用 MCP 工具
func (h *DingTalkMessageHandler) callMCPTool(ctx context.Context, intent *DingTalkIntent) (string, error) {
	// 使用 MCP Server 的公开 CallTool 方法
	result, err := h.mcpServer.CallTool(ctx, intent.MCPTool, intent.Params)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool: %w", err)
	}

	// 提取文本结果
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "查询完成,但未返回结果", nil
}
