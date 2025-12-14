package dingtalk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
)

// CallbackCrypto 回调加解密工具
type CallbackCrypto struct {
	token          string
	encodingAESKey string
	suiteKey       string
	aesKey         []byte
}

// NewCallbackCrypto 创建回调加解密工具
func NewCallbackCrypto(token, encodingAESKey, suiteKey string) (*CallbackCrypto, error) {
	if len(encodingAESKey) != 43 {
		return nil, fmt.Errorf("invalid encoding aes key length: %d", len(encodingAESKey))
	}

	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("failed to decode aes key: %w", err)
	}

	return &CallbackCrypto{
		token:          token,
		encodingAESKey: encodingAESKey,
		suiteKey:       suiteKey,
		aesKey:         aesKey,
	}, nil
}

// CallbackMessage 回调消息结构
type CallbackMessage struct {
	MsgID            string       `json:"msgId"`
	MsgType          string       `json:"msgtype"`
	CreateAt         int64        `json:"createAt"`
	ConversationID   string       `json:"conversationId"`
	ConversationType string       `json:"conversationType"` // "1" 单聊, "2" 群聊
	SenderID         string       `json:"senderId"`
	SenderNick       string       `json:"senderNick"`
	SenderStaffID    string       `json:"senderStaffId"`
	ChatbotUserID    string       `json:"chatbotUserId"`
	RobotCode        string       `json:"robotCode"`
	IsAdmin          bool         `json:"isAdmin"`
	SessionWebhook   string       `json:"sessionWebhook"`
	Text             *TextContent `json:"text,omitempty"`
	AtUsers          []AtUser     `json:"atUsers,omitempty"`
}

// TextContent 文本消息内容
type TextContent struct {
	Content string `json:"content"`
}

// AtUser @的用户
type AtUser struct {
	DingtalkID string `json:"dingtalkId"`
	StaffID    string `json:"staffId"`
}

// CallbackRequest 回调请求
type CallbackRequest struct {
	Encrypt string `json:"encrypt"`
}

// CallbackResponse 回调响应
type CallbackResponse struct {
	MsgType    string         `json:"msgtype"`
	Text       *TextMsg       `json:"text,omitempty"`
	Markdown   *MarkdownMsg   `json:"markdown,omitempty"`
	ActionCard *ActionCardMsg `json:"actionCard,omitempty"`
}

// TextMsg 文本消息
type TextMsg struct {
	Content string `json:"content"`
}

// MarkdownMsg Markdown 消息
type MarkdownMsg struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// ActionCardMsg 卡片消息
type ActionCardMsg struct {
	Title          string `json:"title"`
	Text           string `json:"text"`
	SingleTitle    string `json:"singleTitle,omitempty"`
	SingleURL      string `json:"singleURL,omitempty"`
	BtnOrientation string `json:"btnOrientation,omitempty"`
	Btns           []Btn  `json:"btns,omitempty"`
}

// Btn 卡片按钮
type Btn struct {
	Title     string `json:"title"`
	ActionURL string `json:"actionURL"`
}

// VerifySignature 验证签名
func (c *CallbackCrypto) VerifySignature(timestamp, nonce, body, signature string) bool {
	// 将时间戳转换为字符串
	message := timestamp + "\n" + nonce + "\n" + body

	mac := hmac.New(sha256.New, []byte(c.token))
	mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	logx.Debug("Verifying signature: timestamp %s, nonce %s, expected %s, actual %s",
		timestamp,
		nonce,
		expected,
		signature)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// DecryptMessage 解密消息
func (c *CallbackCrypto) DecryptMessage(encryptedMsg string) (*CallbackMessage, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	iv := c.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// PKCS7 反填充
	plaintext = c.pkcs7UnPadding(plaintext)

	// 去除随机字符串和长度信息
	// 格式: 16位随机字符串 + 4字节消息长度 + 消息内容 + suiteKey
	if len(plaintext) < 20 {
		return nil, fmt.Errorf("plaintext too short")
	}

	// 提取消息长度(大端序)
	msgLen := int(plaintext[16])<<24 | int(plaintext[17])<<16 | int(plaintext[18])<<8 | int(plaintext[19])

	if len(plaintext) < 20+msgLen {
		return nil, fmt.Errorf("invalid message length")
	}

	// 提取消息内容
	msgContent := plaintext[20 : 20+msgLen]

	logx.Debug("Decrypted message: %s", string(msgContent))

	var msg CallbackMessage
	if err := json.Unmarshal(msgContent, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &msg, nil
}

// EncryptMessage 加密消息(用于响应)
func (c *CallbackCrypto) EncryptMessage(msg string) (string, error) {
	// 生成16位随机字符串
	randomStr := c.randomString(16)

	// 消息长度(大端序)
	msgLen := len(msg)
	lengthBytes := []byte{
		byte(msgLen >> 24),
		byte(msgLen >> 16),
		byte(msgLen >> 8),
		byte(msgLen),
	}

	// 拼接: 随机字符串 + 消息长度 + 消息内容 + suiteKey
	plaintext := []byte(randomStr)
	plaintext = append(plaintext, lengthBytes...)
	plaintext = append(plaintext, []byte(msg)...)
	plaintext = append(plaintext, []byte(c.suiteKey)...)

	// PKCS7 填充
	plaintext = c.pkcs7Padding(plaintext, aes.BlockSize)

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	iv := c.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// pkcs7Padding PKCS7 填充
func (c *CallbackCrypto) pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// pkcs7UnPadding PKCS7 反填充
func (c *CallbackCrypto) pkcs7UnPadding(data []byte) []byte {
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

// randomString 生成随机字符串
func (c *CallbackCrypto) randomString(length int) string {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rng.Intn(len(chars))]
	}
	return string(result)
}

// CreateResponse 创建响应消息
func CreateTextResponse(content string) *CallbackResponse {
	return &CallbackResponse{
		MsgType: "text",
		Text: &TextMsg{
			Content: content,
		},
	}
}

// CreateMarkdownResponse 创建 Markdown 响应
func CreateMarkdownResponse(title, text string) *CallbackResponse {
	return &CallbackResponse{
		MsgType: "markdown",
		Markdown: &MarkdownMsg{
			Title: title,
			Text:  text,
		},
	}
}

// CreateActionCardResponse 创建卡片响应
func CreateActionCardResponse(title, text, singleTitle, singleURL string) *CallbackResponse {
	return &CallbackResponse{
		MsgType: "actionCard",
		ActionCard: &ActionCardMsg{
			Title:       title,
			Text:        text,
			SingleTitle: singleTitle,
			SingleURL:   singleURL,
		},
	}
}

// ExtractUserMessage 提取用户消息内容(去除@机器人部分)
func ExtractUserMessage(msg *CallbackMessage) string {
	if msg.Text == nil {
		return ""
	}

	content := msg.Text.Content

	// 去除 @机器人 的内容
	if len(msg.AtUsers) > 0 {
		for _, atUser := range msg.AtUsers {
			if atUser.DingtalkID == msg.ChatbotUserID {
				// 去除 @xxx
				content = strings.TrimSpace(strings.ReplaceAll(content, "@"+msg.ChatbotUserID, ""))
			}
		}
	}

	return strings.TrimSpace(content)
}

// IsValidTimestamp 检查时间戳是否有效(���重放攻击)
func IsValidTimestamp(timestamp string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().UnixMilli()
	diff := now - ts

	// 时间戳在5分钟内有效
	return diff >= 0 && diff <= 5*60*1000
}
