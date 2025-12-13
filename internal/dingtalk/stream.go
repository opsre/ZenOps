package dingtalk

import (
	"context"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
)

// StreamManager 流式消息管理器
type StreamManager struct {
	client *Client
}

// NewStreamManager 创建流式消息管理器
func NewStreamManager(client *Client) *StreamManager {
	return &StreamManager{
		client: client,
	}
}

// Send 发送流式消息
func (s *StreamManager) Send(ctx context.Context, conversationID, streamID, content string, finished bool) error {
	logx.Debug("Sending stream message, conversation_id %s, stream_id %s, content_len %d, finished %t",
		conversationID,
		streamID,
		len(content),
		finished)
	err := s.client.SendStreamMessage(ctx, conversationID, streamID, content, finished)
	if err != nil {
		logx.Error("Failed to send stream message %v", err)
		return err
	}

	// 避免发送过快
	if !finished {
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// SendInChunks 分块发送长消息
func (s *StreamManager) SendInChunks(ctx context.Context, conversationID, streamID, content string) error {
	const chunkSize = 1000 // 每块最大字符数

	if len(content) <= chunkSize {
		// 内容不长,直接发送
		return s.Send(ctx, conversationID, streamID, content, true)
	}

	// 分块发送
	lines := strings.Split(content, "\n")
	var currentChunk strings.Builder
	chunkCount := 0

	for i, line := range lines {
		// 检查当前块是否会超过大小限制
		if currentChunk.Len()+len(line)+1 > chunkSize {
			// 发送当前块
			if currentChunk.Len() > 0 {
				chunkCount++
				if err := s.Send(ctx, conversationID, streamID, currentChunk.String(), false); err != nil {
					return err
				}
				currentChunk.Reset()
			}
		}

		// 添加行到当前块
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)

		// 如果是最后一行,发送最终块
		if i == len(lines)-1 {
			return s.Send(ctx, conversationID, streamID, currentChunk.String(), true)
		}
	}

	return nil
}

// SendProgress 发送进度消息
func (s *StreamManager) SendProgress(ctx context.Context, conversationID, streamID string, progress int, total int, message string) error {
	percentage := 0
	if total > 0 {
		percentage = progress * 100 / total
	}

	content := ""
	if message != "" {
		content = "⏳ " + message + "\n\n"
	}

	content += "进度: " + s.generateProgressBar(percentage) + "\n"
	content += "" + string(rune(progress)) + "/" + string(rune(total))

	return s.Send(ctx, conversationID, streamID, content, false)
}

// generateProgressBar 生成进度条
func (s *StreamManager) generateProgressBar(percentage int) string {
	const barLength = 20
	filled := percentage * barLength / 100
	if filled > barLength {
		filled = barLength
	}

	var bar strings.Builder
	bar.WriteString("[")
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}
	bar.WriteString("] ")
	bar.WriteString(string(rune(percentage)))
	bar.WriteString("%")

	return bar.String()
}

// SendError 发送错误消息
func (s *StreamManager) SendError(ctx context.Context, conversationID, streamID string, err error) error {
	content := "❌ **操作失败**\n\n"
	content += "错误信息: " + err.Error() + "\n\n"
	content += "💡 请检查参数后重试,或发送\"帮助\"查看使用说明"

	return s.Send(ctx, conversationID, streamID, content, true)
}

// SendSuccess 发送成功消息
func (s *StreamManager) SendSuccess(ctx context.Context, conversationID, streamID, message string) error {
	content := "✅ **操作成功**\n\n" + message
	return s.Send(ctx, conversationID, streamID, content, true)
}

// SendTable 发送表格数据
func (s *StreamManager) SendTable(ctx context.Context, conversationID, streamID string, headers []string, rows [][]string) error {
	var content strings.Builder

	// 构建 Markdown 表格
	// 表头
	content.WriteString("| ")
	for _, h := range headers {
		content.WriteString(h)
		content.WriteString(" | ")
	}
	content.WriteString("\n")

	// 分隔线
	content.WriteString("|")
	for range headers {
		content.WriteString(" --- |")
	}
	content.WriteString("\n")

	// 数据行
	for _, row := range rows {
		content.WriteString("| ")
		for _, cell := range row {
			content.WriteString(cell)
			content.WriteString(" | ")
		}
		content.WriteString("\n")
	}

	return s.SendInChunks(ctx, conversationID, streamID, content.String())
}

// SendList 发送列表数据
func (s *StreamManager) SendList(ctx context.Context, conversationID, streamID string, items []string) error {
	var content strings.Builder

	for i, item := range items {
		content.WriteString(string(rune(i + 1)))
		content.WriteString(". ")
		content.WriteString(item)
		content.WriteString("\n")
	}

	return s.SendInChunks(ctx, conversationID, streamID, content.String())
}
