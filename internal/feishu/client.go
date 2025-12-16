package feishu

import (
	"context"
	"encoding/json"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkcardkit "github.com/larksuite/oapi-sdk-go/v3/service/cardkit/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Client 飞书客户端
type Client struct {
	appID     string
	appSecret string
	client    *lark.Client
}

// FeishuLogger 飞书日志适配器
type FeishuLogger struct{}

func (l *FeishuLogger) Debug(ctx context.Context, args ...interface{}) {
	logx.Debug("feishu: %v", args)
}

func (l *FeishuLogger) Info(ctx context.Context, args ...interface{}) {
	logx.Info("feishu: %v", args)
}

func (l *FeishuLogger) Warn(ctx context.Context, args ...interface{}) {
	logx.Warn("feishu: %v", args)
}

func (l *FeishuLogger) Error(ctx context.Context, args ...interface{}) {
	logx.Error("feishu: %v", args)
}

// NewClient 创建飞书客户端
func NewClient(appID, appSecret string) *Client {
	client := lark.NewClient(appID, appSecret,
		lark.WithLogLevel(larkcore.LogLevelInfo),
		lark.WithLogger(&FeishuLogger{}),
	)

	logx.Info("Feishu client created, app_id %s", appID)

	return &Client{
		appID:     appID,
		appSecret: appSecret,
		client:    client,
	}
}

// SendTextMessage 发送文本消息
func (c *Client) SendTextMessage(ctx context.Context, receiveIDType, receiveID, text string) error {
	content := fmt.Sprintf(`{"text":"%s"}`, text)

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType("text").
			Content(content).
			Build()).
		Build()

	resp, err := c.client.Im.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send text message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("failed to send text message: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	logx.Debug("Sent text message to %s: %s", receiveID, text)
	return nil
}

// SendMarkdownMessage 发送富文本消息(支持 Markdown)
func (c *Client) SendMarkdownMessage(ctx context.Context, receiveIDType, receiveID, title, content string) (*string, error) {
	// 构建富文本内容 - 使用 post 类型支持 markdown
	postContent := map[string]interface{}{
		"zh_cn": map[string]interface{}{
			"title": title,
			"content": [][]map[string]interface{}{
				{
					{
						"tag":  "text",
						"text": content,
					},
				},
			},
		},
	}

	contentJSON, err := json.Marshal(postContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType("post").
			Content(string(contentJSON)).
			Build()).
		Build()

	resp, err := c.client.Im.Message.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send markdown message: %w", err)
	}

	if !resp.Success() {
		return nil, fmt.Errorf("failed to send markdown message: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	logx.Debug("Sent markdown message to %s", receiveID)
	return resp.Data.MessageId, nil
}

// CreateStreamingCard 创建流式卡片
func (c *Client) CreateStreamingCard(ctx context.Context, title, initialContent string) (string, error) {
	// 构建流式卡片模板
	cardTemplate := map[string]interface{}{
		"schema": "2.0",
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": title,
				"tag":     "plain_text",
			},
		},
		"config": map[string]interface{}{
			"streaming_mode": true,
			"summary": map[string]interface{}{
				"content": "",
			},
		},
		"body": map[string]interface{}{
			"elements": []map[string]interface{}{
				{
					"tag":        "markdown",
					"content":    initialContent,
					"element_id": "markdown_content",
				},
			},
		},
	}

	cardData, err := json.Marshal(cardTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to marshal card template: %w", err)
	}

	// 创建卡片
	req := larkcardkit.NewCreateCardReqBuilder().
		Body(larkcardkit.NewCreateCardReqBodyBuilder().
			Type("card_json").
			Data(string(cardData)).
			Build()).
		Build()

	resp, err := c.client.Cardkit.V1.Card.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create card: %w", err)
	}

	if !resp.Success() {
		return "", fmt.Errorf("failed to create card: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	logx.Debug("Created streaming card: %s", *resp.Data.CardId)
	return *resp.Data.CardId, nil
}

// SendCardMessage 发送卡片消息
func (c *Client) SendCardMessage(ctx context.Context, receiveIDType, receiveID, cardID string) (*string, error) {
	// 构建卡片消息内容
	cardContent := map[string]interface{}{
		"type": "card",
		"data": map[string]string{
			"card_id": cardID,
		},
	}

	contentJSON, err := json.Marshal(cardContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal card content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType("interactive").
			Content(string(contentJSON)).
			Build()).
		Build()

	resp, err := c.client.Im.Message.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send card message: %w", err)
	}

	if !resp.Success() {
		return nil, fmt.Errorf("failed to send card message: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	logx.Debug("Sent card message to %s", receiveID)
	return resp.Data.MessageId, nil
}

// UpdateCardElement 更新卡片元素内容
func (c *Client) UpdateCardElement(ctx context.Context, cardID, elementID, content string, sequence int) error {
	// 生成唯一的 UUID
	uuid := fmt.Sprintf("%d-%d", ctx.Value("timestamp"), sequence)

	req := larkcardkit.NewContentCardElementReqBuilder().
		CardId(cardID).
		ElementId(elementID).
		Body(larkcardkit.NewContentCardElementReqBodyBuilder().
			Uuid(uuid).
			Content(content).
			Sequence(sequence).
			Build()).
		Build()

	resp, err := c.client.Cardkit.V1.CardElement.Content(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update card element: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("failed to update card element: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	logx.Debug("Updated card element %s, sequence %d", elementID, sequence)
	return nil
}

// SendInteractiveCard 发送交互式卡片
func (c *Client) SendInteractiveCard(ctx context.Context, receiveIDType, receiveID, cardContent string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType("interactive").
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send interactive card: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("failed to send interactive card: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	logx.Debug("Sent interactive card to %s", receiveID)
	return nil
}

// escapeMarkdown 转义 Markdown 特殊字符
func escapeMarkdown(s string) string {
	// 简单的转义,实际使用中可能需要更完善的处理
	replacer := map[rune]string{
		'"':  `\"`,
		'\\': `\\`,
		'\n': `\n`,
		'\r': ``,
	}

	result := ""
	for _, c := range s {
		if replacement, ok := replacer[c]; ok {
			result += replacement
		} else {
			result += string(c)
		}
	}
	return result
}
