package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
)

// Client 钉钉客户端
type Client struct {
	appKey      string
	appSecret   string
	agentID     string
	accessToken string
	tokenMutex  sync.RWMutex
	tokenExpire time.Time
	httpClient  *http.Client
}

// NewClient 创建钉钉客户端
func NewClient(appKey, appSecret, agentID string) *Client {
	return &Client{
		appKey:    appKey,
		appSecret: appSecret,
		agentID:   agentID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AccessTokenResponse 获取 AccessToken 的响应
type AccessTokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetAccessToken 获取访问令牌
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpire) {
		token := c.accessToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// 双重检查
	if c.accessToken != "" && time.Now().Before(c.tokenExpire) {
		return c.accessToken, nil
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/gettoken?appkey=%s&appsecret=%s", c.appKey, c.appSecret)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("dingtalk api error: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpire = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second) // 提前5分钟过期

	logx.Debug("Got DingTalk access token: %s, expire: %s",
		c.accessToken[:20]+"...",
		c.tokenExpire)

	return c.accessToken, nil
}

// SendTextMessage 发送文本消息
func (c *Client) SendTextMessage(ctx context.Context, conversationID, content string) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", token)

	message := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	return c.sendRequest(ctx, url, message)
}

// SendMarkdownMessage 发送 Markdown 消息
func (c *Client) SendMarkdownMessage(ctx context.Context, conversationID, title, text string) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", token)

	message := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}

	return c.sendRequest(ctx, url, message)
}

// SendStreamMessage 发送流式消息
func (c *Client) SendStreamMessage(ctx context.Context, conversationID, streamID, content string, finished bool) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/chat/send/stream?access_token=%s", token)

	message := map[string]any{
		"conversation_id": conversationID,
		"stream_id":       streamID,
		"content":         content,
		"finished":        finished,
	}

	return c.sendRequest(ctx, url, message)
}

// SendInteractiveCardMessage 发送交互式卡片消息
func (c *Client) SendInteractiveCardMessage(ctx context.Context, conversationID string, card any) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/topapi/im/chat/scenegroup/interactivecard/send?access_token=%s", token)

	message := map[string]any{
		"conversation_id": conversationID,
		"card_data":       card,
	}

	return c.sendRequest(ctx, url, message)
}

// sendRequest 发送 HTTP 请求
func (c *Client) sendRequest(ctx context.Context, url string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	logx.Debug("Sending DingTalk request: url %s, payload %s",
		url,
		string(data))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	logx.Debug("DingTalk response: %v", result)

	if errCode, ok := result["errcode"].(float64); ok && errCode != 0 {
		return fmt.Errorf("dingtalk api error: %v - %v", result["errcode"], result["errmsg"])
	}

	return nil
}

// GetUserInfo 获取用户信息
func (c *Client) GetUserInfo(ctx context.Context, userID string) (map[string]any, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/topapi/v2/user/get?access_token=%s", token)

	payload := map[string]any{
		"userid": userID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if errCode, ok := result["errcode"].(float64); ok && errCode != 0 {
		return nil, fmt.Errorf("dingtalk api error: %v - %v", result["errcode"], result["errmsg"])
	}

	if userInfo, ok := result["result"].(map[string]any); ok {
		return userInfo, nil
	}

	return nil, fmt.Errorf("invalid user info response")
}
