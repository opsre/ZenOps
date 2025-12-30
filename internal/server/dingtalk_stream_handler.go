package server

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/llm"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
)

// Intent 用户意图
type Intent struct {
	Action   string            // list, get, search
	Provider string            // aliyun, tencent, jenkins
	Resource string            // ecs, cvm, rds, cdb, job, build
	Params   map[string]string // 参数
	MCPTool  string            // 对应的 MCP 工具名称
}

// IntentParser 意图解析器
type IntentParser struct {
	patterns []intentPattern
}

type intentPattern struct {
	regex     *regexp.Regexp
	provider  string
	resource  string
	action    string
	extractor func([]string) map[string]string
}

// DingTalkStreamHandler Stream模式处理器
type DingTalkStreamHandler struct {
	config         *config.Config
	cardClient     *DingTalkStreamClient
	mcpServer      *imcp.MCPServer
	streamClient   *client.StreamClient
	intentParser   *IntentParser
	llmClient      *llm.Client
	chatLogService *service.ChatLogService
}

// NewDingTalkStreamHandler 创建Stream处理器
func NewDingTalkStreamHandler(cfg *config.Config, cardClient *DingTalkStreamClient, mcpServer *imcp.MCPServer) *DingTalkStreamHandler {
	handler := &DingTalkStreamHandler{
		config:         cfg,
		cardClient:     cardClient,
		mcpServer:      mcpServer,
		intentParser:   newIntentParser(),
		chatLogService: service.NewChatLogService(),
	}

	// 初始化 LLM 客户端
	// 优先从数据库读取 LLM 配置，如果数据库没有配置则使用 config.yaml
	configService := service.NewConfigService()
	dbLLMConfig, err := configService.GetDefaultLLMConfig()

	var llmEnabled bool
	var llmCfg *llm.Config

	if err == nil && dbLLMConfig != nil && dbLLMConfig.Enabled {
		// 使用数据库配置
		llmEnabled = true
		llmCfg = &llm.Config{
			Model:   dbLLMConfig.Model,
			APIKey:  dbLLMConfig.APIKey,
			BaseURL: dbLLMConfig.BaseURL,
		}
		logx.Info("⚗️ Using LLM Config from Database: %s (Model: %s)", dbLLMConfig.Name, dbLLMConfig.Model)
	} else if cfg.LLM.Enabled {
		// 降级使用 config.yaml 配置
		llmEnabled = true
		llmCfg = &llm.Config{
			Model:   cfg.LLM.Model,
			APIKey:  cfg.LLM.APIKey,
			BaseURL: cfg.LLM.BaseURL,
		}
		logx.Info("⚗️ Using LLM Config from config.yaml (Model: %s)", cfg.LLM.Model)
	}

	if llmEnabled {
		handler.llmClient = llm.NewClient(llmCfg, mcpServer)
		logx.Info("⚗️ LLM Client Initialized For DingTalk Stream Handler")
	}

	return handler
}

// Start 启动Stream客户端
func (h *DingTalkStreamHandler) Start(ctx context.Context) error {
	// 创建Stream客户端
	h.streamClient = client.NewStreamClient(client.WithAppCredential(
		client.NewAppCredentialConfig(h.config.DingTalk.AppKey, h.config.DingTalk.AppSecret),
	))

	// 注册机器人回调处理器
	h.streamClient.RegisterChatBotCallbackRouter(h.onChatBotMessage)

	// 启动客户端
	return h.streamClient.Start(ctx)
}

// Stop 停止Stream客户端
func (h *DingTalkStreamHandler) Stop() error {
	if h.streamClient != nil {
		logx.Info("Stopping DingTalk Stream client")
		h.streamClient.Close()
	}
	return nil
}

// onChatBotMessage 处理机器人消息回调
func (h *DingTalkStreamHandler) onChatBotMessage(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	logx.Info("Received chatbot message from %s in conversation %s", data.SenderNick, data.ConversationId)

	// 提取消息内容
	content := data.Text.Content

	// 去除@机器人的部分
	content = h.cleanAtMention(content, data.ChatbotUserId, data.AtUsers)

	logx.Debug("Parsed message, content=%s", content)

	// 帮助命令
	if strings.Contains(content, "帮助") || strings.Contains(content, "help") {
		h.sendHelpMessage(ctx, data)
		return []byte(""), nil
	}

	// 如果启用了 LLM,使用 LLM 处理
	if h.config.LLM.Enabled && h.llmClient != nil {
		logx.Info("Using LLM to process message")
		go h.processLLMMessage(ctx, data, content)
		return []byte(""), nil
	}

	// 传统的意图解析模式
	intent, err := h.intentParser.Parse(content)
	if err != nil {
		h.sendErrorMessage(ctx, data, content, err)
		return []byte(""), nil
	}

	// 异步处理查询
	go h.processQueryAsync(ctx, data, content, intent)

	return []byte(""), nil
}

// cleanAtMention 清理消息中的@提及
func (h *DingTalkStreamHandler) cleanAtMention(content, chatbotUserID string, atUsers []chatbot.BotCallbackDataAtUserModel) string {
	// 去除@机器人
	content = strings.ReplaceAll(content, "@"+chatbotUserID, "")

	// 去除其他@用户
	for _, user := range atUsers {
		if user.DingtalkId != "" {
			content = strings.ReplaceAll(content, "@"+user.DingtalkId, "")
		}
	}

	return strings.TrimSpace(content)
}

// sendHelpMessage 发送帮助信息
func (h *DingTalkStreamHandler) sendHelpMessage(ctx context.Context, data *chatbot.BotCallbackDataModel) {
	helpContent := getHelpMessage()

	// 确定消息来源（私聊/群聊）
	source := "私聊"
	if data.ConversationType == "2" {
		source = "群聊"
	}

	// 保存用户消息到数据库
	username := data.SenderNick
	if username == "" {
		username = data.SenderStaffId
	}
	userLog, err := h.chatLogService.CreateUserMessage(username, source, "帮助")
	if err != nil {
		logx.Error("Failed to save user message to database: %v", err)
	}

	// 检查是否配置了卡片模板ID
	if h.config.DingTalk.CardTemplateID == "" {
		// 使用传统文本回复
		logx.Debug("Card template not configured, using text reply")
		h.sendTextReply(data, helpContent)
		// 保存帮助消息到数据库
		if userLog != nil {
			_, saveErr := h.chatLogService.CreateAIMessage(username, source, helpContent, userLog.ID)
			if saveErr != nil {
				logx.Error("Failed to save help message to database: %v", saveErr)
			}
		}
		return
	}

	trackID := h.generateTrackID(data.MsgId)

	// 创建卡片
	if err := h.createCard(ctx, trackID, data); err != nil {
		logx.Error("Failed to create help card, fallback to text: %v", err)
		h.sendTextReply(data, helpContent)
		// 保存帮助消息到数据库
		if userLog != nil {
			_, saveErr := h.chatLogService.CreateAIMessage(username, source, helpContent, userLog.ID)
			if saveErr != nil {
				logx.Error("Failed to save help message to database: %v", saveErr)
			}
		}
		return
	}

	_ = h.cardClient.StreamingUpdate(trackID, helpContent, true)

	// 保存帮助消息到数据库
	if userLog != nil {
		_, err := h.chatLogService.CreateAIMessage(username, source, helpContent, userLog.ID)
		if err != nil {
			logx.Error("Failed to save help message to database: %v", err)
		}
	}
}

// sendErrorMessage 发送错误消息
func (h *DingTalkStreamHandler) sendErrorMessage(ctx context.Context, data *chatbot.BotCallbackDataModel, question string, err error) {
	errorContent := fmt.Sprintf(`❌ 无法理解您的请求

错误: %s

💡 您可以发送 "帮助" 查看支持的命令`, err.Error())

	// 检查是否配置了卡片模板ID
	if h.config.DingTalk.CardTemplateID == "" {
		// 使用传统文本回复
		logx.Debug("Card template not configured, using text reply")
		h.sendTextReply(data, errorContent)
		return
	}

	trackID := h.generateTrackID(data.MsgId)

	// 创建卡片
	if createErr := h.createCard(ctx, trackID, data); createErr != nil {
		logx.Error("Failed to create error card, fallback to text: %v", createErr)
		h.sendTextReply(data, errorContent)
		return
	}

	_ = h.cardClient.StreamingUpdate(trackID, errorContent, true)
}

// processQueryAsync 异步处理查询
func (h *DingTalkStreamHandler) processQueryAsync(ctx context.Context, data *chatbot.BotCallbackDataModel, question string, intent *Intent) {
	// 检查是否配置了卡片模板ID
	useCard := h.config.DingTalk.CardTemplateID != ""

	var trackID string
	if useCard {
		trackID = h.generateTrackID(data.MsgId)

		// 1. 创建并投递AI卡片
		if err := h.createCard(ctx, trackID, data); err != nil {
			logx.Error("Failed to create card, fallback to text reply: %v", err)
			useCard = false
		}
	}

	if !useCard {
		// 使用传统文本回复,先发送一个"正在查询"的消息
		h.sendTextReply(data, fmt.Sprintf("🔍 正在查询 %s %s,请稍候...",
			h.getProviderName(intent.Provider),
			h.getResourceName(intent.Resource)))
	}

	// 2. 如果使用卡片,发送初始提示
	if useCard {
		initialContent := fmt.Sprintf("**%s**\n\n⏳ 正在查询 %s %s...",
			question,
			h.getProviderName(intent.Provider),
			h.getResourceName(intent.Resource))

		if err := h.cardClient.StreamingUpdate(trackID, initialContent, false); err != nil {
			logx.Error("Failed to send initial message: %v", err)
		}
	}

	// 3. 调用MCP工具
	result, err := h.callMCPTool(ctx, intent)
	if err != nil {
		logx.Error("Failed to call MCP tool: %v", err)

		if useCard {
			errorContent := fmt.Sprintf("**%s**\n\n❌ **查询失败**\n\n错误: %s", question, err.Error())
			_ = h.cardClient.StreamingUpdate(trackID, errorContent, true)
		} else {
			h.sendTextReply(data, fmt.Sprintf("❌ 查询失败\n\n错误: %s", err.Error()))
		}
		return
	}

	// 4. 发送结果
	if useCard {
		h.streamResult(ctx, trackID, question, intent, result)
	} else {
		// 使用文本回复发送结果
		formattedResult := fmt.Sprintf("✅ **%s %s 查询完成**\n\n%s",
			h.getProviderName(intent.Provider),
			h.getResourceName(intent.Resource),
			result)
		h.sendTextReply(data, formattedResult)
	}
}

// createCard 创建AI卡片
func (h *DingTalkStreamHandler) createCard(ctx context.Context, trackID string, data *chatbot.BotCallbackDataModel) error {
	return h.cardClient.CreateAndDeliverCard(ctx, trackID, data.ConversationId, data.ConversationType, data.SenderStaffId)
}

// streamResult 流式发送结果
func (h *DingTalkStreamHandler) streamResult(ctx context.Context, trackID, question string, intent *Intent, result string) {
	// 格式化结果头部
	header := fmt.Sprintf("**%s**\n\n✅ **%s %s 查询完成**\n\n",
		question,
		h.getProviderName(intent.Provider),
		h.getResourceName(intent.Resource))

	// 分行流式发送
	lines := strings.Split(result, "\n")
	currentContent := header

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for i, line := range lines {
		currentContent += line + "\n"

		// 每5行或最后一行更新一次
		if (i+1)%5 == 0 || i == len(lines)-1 {
			isFinalize := i == len(lines)-1

			// 如果是最后一行,添加时间戳
			if isFinalize {
				currentContent += fmt.Sprintf("\n---\n⏰ 查询时间: %s",
					time.Now().Format("2006-01-02 15:04:05"))
			}

			if err := h.cardClient.StreamingUpdate(trackID, currentContent, isFinalize); err != nil {
				logx.Error("Failed to update card: %v", err)
				break
			}

			if !isFinalize {
				<-ticker.C // 等待一段时间再更新
			}
		}
	}
}

// callMCPTool 调用MCP工具
func (h *DingTalkStreamHandler) callMCPTool(ctx context.Context, intent *Intent) (string, error) {
	logx.Debug("Calling MCP tool, tool %s, params %v", intent.MCPTool, intent.Params)

	// 转换参数
	params := make(map[string]any)
	for k, v := range intent.Params {
		params[k] = v
	}

	// 使用MCP Server的CallTool方法
	result, err := h.mcpServer.CallTool(ctx, intent.MCPTool, params)
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

// getProviderName 获取云平台名称
func (h *DingTalkStreamHandler) getProviderName(provider string) string {
	names := map[string]string{
		"aliyun":  "阿里云",
		"tencent": "腾讯云",
		"jenkins": "Jenkins",
	}
	if name, ok := names[provider]; ok {
		return name
	}
	return provider
}

// getResourceName 获取资源名称
func (h *DingTalkStreamHandler) getResourceName(resource string) string {
	names := map[string]string{
		"ecs":   "ECS",
		"rds":   "RDS",
		"cvm":   "CVM",
		"cdb":   "CDB",
		"job":   "Job",
		"build": "Build",
	}
	if name, ok := names[resource]; ok {
		return name
	}
	return resource
}

// generateTrackID 生成跟踪ID
func (h *DingTalkStreamHandler) generateTrackID(msgID string) string {
	return fmt.Sprintf("track_%s_%s", msgID, uuid.New().String()[:8])
}

// newIntentParser 创建意图解析器
func newIntentParser() *IntentParser {
	parser := &IntentParser{
		patterns: make([]intentPattern, 0),
	}
	parser.registerPatterns()
	return parser
}

// registerPatterns 注册意图匹配模式
func (p *IntentParser) registerPatterns() {
	// ==================== 阿里云 ECS ====================

	// 按 IP 搜索 ECS
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|找|搜索?)(一?下?)?.*(阿里云?)?.*(IP|ip).*([\d\.]+)`),
		provider: "aliyun",
		resource: "ecs",
		action:   "search_ip",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"ip": matches[5]}
		},
	})

	// 按名称搜索 ECS
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|找|搜索?)(一?下?)?.*(阿里云?)?.*(名称?|名字|叫).*([\w\-]+)`),
		provider: "aliyun",
		resource: "ecs",
		action:   "search_name",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"name": matches[5]}
		},
	})

	// 列出 ECS 实例
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(列出|查询?|看).*(阿里云?).*(ECS|ecs|服务器|实例)`),
		provider: "aliyun",
		resource: "ecs",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			params := make(map[string]string)
			if strings.Contains(matches[0], "杭州") {
				params["region"] = "cn-hangzhou"
			} else if strings.Contains(matches[0], "上海") {
				params["region"] = "cn-shanghai"
			} else if strings.Contains(matches[0], "北京") {
				params["region"] = "cn-beijing"
			}
			return params
		},
	})

	// ==================== 阿里云 RDS ====================

	// 列出 RDS 实例
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(列出|查询?|看).*(阿里云?).*(RDS|rds|数据库)`),
		provider: "aliyun",
		resource: "rds",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			params := make(map[string]string)
			if strings.Contains(matches[0], "杭州") {
				params["region"] = "cn-hangzhou"
			} else if strings.Contains(matches[0], "上海") {
				params["region"] = "cn-shanghai"
			}
			return params
		},
	})

	// 按名称搜索 RDS
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|找|搜索?).*(RDS|rds|数据库).*(名称?|名字|叫).*([\w\-]+)`),
		provider: "aliyun",
		resource: "rds",
		action:   "search_name",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"name": matches[4]}
		},
	})

	// ==================== 腾讯云 CVM ====================

	// 按 IP 搜索 CVM
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|找|搜索?)(一?下?)?.*(腾讯云?).*(IP|ip).*([\d\.]+)`),
		provider: "tencent",
		resource: "cvm",
		action:   "search_ip",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"ip": matches[5]}
		},
	})

	// 按名称搜索 CVM
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|找|搜索?)(一?下?)?.*(腾讯云?).*(名称?|名字|叫).*([\w\-]+)`),
		provider: "tencent",
		resource: "cvm",
		action:   "search_name",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"name": matches[5]}
		},
	})

	// 列出 CVM 实例
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(列出|查询?|看).*(腾讯云?).*(CVM|cvm|服务器|实例)`),
		provider: "tencent",
		resource: "cvm",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			params := make(map[string]string)
			if strings.Contains(matches[0], "广州") {
				params["region"] = "ap-guangzhou"
			} else if strings.Contains(matches[0], "上海") {
				params["region"] = "ap-shanghai"
			} else if strings.Contains(matches[0], "北京") {
				params["region"] = "ap-beijing"
			}
			return params
		},
	})

	// ==================== 腾讯云 CDB ====================

	// 列出 CDB 实例
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(列出|查询?|看).*(腾讯云?).*(CDB|cdb|数据库)`),
		provider: "tencent",
		resource: "cdb",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			params := make(map[string]string)
			if strings.Contains(matches[0], "广州") {
				params["region"] = "ap-guangzhou"
			}
			return params
		},
	})

	// 按名称搜索 CDB
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|找|搜索?).*(CDB|cdb|数据库).*(名称?|名字|叫).*([\w\-]+)`),
		provider: "tencent",
		resource: "cdb",
		action:   "search_name",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"name": matches[4]}
		},
	})

	// ==================== Jenkins ====================

	// 列出 Jenkins Job
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(列出|查询?|看).*(jenkins|Jenkins).*(job|Job|任务)`),
		provider: "jenkins",
		resource: "job",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			return make(map[string]string)
		},
	})

	// 获取 Job 详情
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(查询?|看).*(job|Job|任务).*([\w\-]+).*(详情|信息)`),
		provider: "jenkins",
		resource: "job",
		action:   "get",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"job_name": matches[3]}
		},
	})

	// 列出构建历史
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(看|查).*([\w\-]+).*(任务|job).*(构建|build|历史)`),
		provider: "jenkins",
		resource: "build",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			return map[string]string{"job_name": matches[2]}
		},
	})

	// 通用 Jenkins 查询
	p.patterns = append(p.patterns, intentPattern{
		regex:    regexp.MustCompile(`(?i)(jenkins|Jenkins)`),
		provider: "jenkins",
		resource: "job",
		action:   "list",
		extractor: func(matches []string) map[string]string {
			return make(map[string]string)
		},
	})
}

// Parse 解析用户消息
func (p *IntentParser) Parse(message string) (*Intent, error) {
	logx.Debug("Parsing intent, message %s", message)

	// 遍历所有模式
	for _, pattern := range p.patterns {
		if matches := pattern.regex.FindStringSubmatch(message); matches != nil {
			logx.Debug("Pattern matched, pattern %s, matches %v", pattern.regex.String(), matches)

			intent := &Intent{
				Provider: pattern.provider,
				Resource: pattern.resource,
				Action:   pattern.action,
				Params:   pattern.extractor(matches),
			}

			// 映射到 MCP 工具
			intent.MCPTool = mapToMCPTool(intent)

			logx.Info("Intent parsed, provider %s, resource %s, action %s, mcp_tool %s, params %v", intent.Provider, intent.Resource, intent.Action, intent.MCPTool, intent.Params)

			return intent, nil
		}
	}

	return nil, fmt.Errorf("无法识别您的请求,请尝试更明确的描述")
}

// mapToMCPTool 将意图映射到 MCP 工具
func mapToMCPTool(intent *Intent) string {
	key := fmt.Sprintf("%s_%s_%s", intent.Provider, intent.Resource, intent.Action)

	mapping := map[string]string{
		// 阿里云 ECS
		"aliyun_ecs_search_ip":   "search_ecs_by_ip",
		"aliyun_ecs_search_name": "search_ecs_by_name",
		"aliyun_ecs_list":        "list_ecs",
		"aliyun_ecs_get":         "get_ecs",

		// 阿里云 RDS
		"aliyun_rds_list":        "list_rds",
		"aliyun_rds_search_name": "search_rds_by_name",

		// 腾讯云 CVM
		"tencent_cvm_search_ip":   "search_cvm_by_ip",
		"tencent_cvm_search_name": "search_cvm_by_name",
		"tencent_cvm_list":        "list_cvm",
		"tencent_cvm_get":         "get_cvm",

		// 腾讯云 CDB
		"tencent_cdb_list":        "list_cdb",
		"tencent_cdb_search_name": "search_cdb_by_name",

		// Jenkins
		"jenkins_job_list":   "list_jenkins_jobs",
		"jenkins_job_get":    "get_jenkins_job",
		"jenkins_build_list": "list_jenkins_builds",
	}

	if tool, ok := mapping[key]; ok {
		return tool
	}

	return ""
}

// getHelpMessage 获取帮助消息
func getHelpMessage() string {
	return `👋 你好!我是 ZenOps 运维助手,可以帮你查询云资源和 CI/CD 信息。

**支持的查询:**

📦 **阿里云**
• 列出 ECS 实例: "查询阿里云杭州的 ECS"
• 搜索 IP: "找一下 IP 为 192.168.1.1 的服务器"
• 搜索名称: "查询名为 web-server 的实例"
• 数据库: "列出阿里云 RDS 数据库"

📦 **腾讯云**
• 列出 CVM: "查询腾讯云广州的 CVM"
• 搜索 IP: "找腾讯云 IP 10.0.0.1 的机器"
• 数据库: "列出腾讯云 CDB"

🔧 **Jenkins**
• 列出任务: "看一下 Jenkins 任务列表"
• 构建历史: "查询 deploy-prod 的构建历史"

**提示:**
• 可以在群里 @我 或私聊我
• 描述越详细,查询越准确
• 支持中文和英文关键词`
}

// sendTextReply 发送文本回复(用于不使用卡片时的降级方案)
func (h *DingTalkStreamHandler) sendTextReply(data *chatbot.BotCallbackDataModel, content string) {
	replier := chatbot.NewChatbotReplier()

	// 构建Markdown消息
	markdownMsg := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "ZenOps 查询结果",
			"text":  content,
		},
	}

	msgBytes, err := json.Marshal(markdownMsg)
	if err != nil {
		logx.Error("Failed to marshal message: %v", err)
		return
	}

	// 发送消息
	err = replier.SimpleReplyMarkdown(
		context.Background(),
		data.SessionWebhook,
		[]byte(""), // atUserIds
		msgBytes,   // message content
	)

	if err != nil {
		logx.Error("Failed to send text reply: %v", err)
		return
	}

	logx.Debug("Sent text reply successfully")
}

// processLLMMessage 使用 LLM 处理消息
func (h *DingTalkStreamHandler) processLLMMessage(ctx context.Context, data *chatbot.BotCallbackDataModel, userMessage string) {
	logx.Info("Processing message with LLM, user %s asked: %s", data.SenderNick, userMessage)

	// 确定消息来源（私聊/群聊）
	source := "私聊"
	if data.ConversationType == "2" {
		source = "群聊"
	}

	// 保存用户消息到数据库
	username := data.SenderNick
	if username == "" {
		username = data.SenderStaffId
	}
	userLog, err := h.chatLogService.CreateUserMessage(username, source, userMessage)
	if err != nil {
		logx.Error("Failed to save user message to database: %v", err)
	}

	// 检查是否使用卡片
	useCard := h.config.DingTalk.CardTemplateID != ""
	var trackID string

	if useCard {
		trackID = h.generateTrackID(data.MsgId)
		// 创建卡片
		if err := h.createCard(ctx, trackID, data); err != nil {
			logx.Error("Failed to create card for LLM, fallback to text: %v", err)
			useCard = false
		}
	}

	// 发送初始消息
	if useCard {
		initialContent := fmt.Sprintf("**%s**\n\n🤖 正在思考...", userMessage)
		if err := h.cardClient.StreamingUpdate(trackID, initialContent, false); err != nil {
			logx.Warn("Failed to send initial message: %v", err)
		}
	} else {
		h.sendTextReply(data, "🤖 正在思考,请稍候...")
	}

	// 调用 LLM
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	if err != nil {
		logx.Error("Failed to call LLM: %v", err)
		errorMsg := fmt.Sprintf("❌ LLM 调用失败: %v", err)

		if useCard {
			_ = h.cardClient.StreamingUpdate(trackID, fmt.Sprintf("**%s**\n\n%s", userMessage, errorMsg), true)
		} else {
			h.sendTextReply(data, errorMsg)
		}
		return
	}

	// 流式接收响应
	if useCard {
		h.streamLLMResponseWithCard(ctx, trackID, userMessage, username, source, userLog, responseCh)
	} else {
		h.streamLLMResponseWithText(data, userMessage, username, source, userLog, responseCh)
	}
}

// streamLLMResponseWithCard 使用卡片流式显示 LLM 响应
func (h *DingTalkStreamHandler) streamLLMResponseWithCard(ctx context.Context, trackID, question, username, source string, userLog *model.ChatLog, responseCh <-chan string) {
	questionHeader := fmt.Sprintf("**%s**\n\n", question)
	fullContent := questionHeader

	// 用于收集AI响应（不包含header）
	var aiResponse strings.Builder

	// 改进的缓冲机制
	updateBuffer := ""
	minUpdateInterval := 200 * time.Millisecond // 减少到200ms,提升响应速度
	minBufferSize := 10                         // 至少累积10个字符再更新

	ticker := time.NewTicker(minUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case content, ok := <-responseCh:
			if !ok {
				// 流结束,发送最终更新
				if updateBuffer != "" {
					fullContent += updateBuffer
					aiResponse.WriteString(updateBuffer)
				}
				fullContent += fmt.Sprintf("\n\n---\n⏰ %s", time.Now().Format("2006-01-02 15:04:05"))

				if err := h.cardClient.StreamingUpdate(trackID, fullContent, true); err != nil {
					logx.Error("Failed to finalize card: %v", err)
				}

				// 保存AI响应到数据库
				if userLog != nil && aiResponse.Len() > 0 {
					_, err := h.chatLogService.CreateAIMessage(username, source, aiResponse.String(), userLog.ID)
					if err != nil {
						logx.Error("Failed to save AI response to database: %v", err)
					}
				}

				logx.Info("LLM conversation completed with card")
				return
			}

			// 累积到缓冲区
			updateBuffer += content

		case <-ticker.C:
			// 定时检查是否需要更新
			if updateBuffer != "" && len(updateBuffer) >= minBufferSize {
				fullContent += updateBuffer
				aiResponse.WriteString(updateBuffer)
				updateBuffer = ""

				// 更新卡片
				if err := h.cardClient.StreamingUpdate(trackID, fullContent, false); err != nil {
					logx.Warn("Failed to update card: %v", err)
				}
			}
		}
	}
}

// streamLLMResponseWithText 使用文本消息显示 LLM 响应
func (h *DingTalkStreamHandler) streamLLMResponseWithText(data *chatbot.BotCallbackDataModel, question, username, source string, userLog *model.ChatLog, responseCh <-chan string) {
	// 累积所有响应
	var fullResponse strings.Builder

	for content := range responseCh {
		fullResponse.WriteString(content)
	}

	aiResponseStr := fullResponse.String()

	// 格式化并发送完整响应
	result := fmt.Sprintf("**问题:** %s\n\n**回答:**\n\n%s\n\n---\n⏰ %s",
		question,
		aiResponseStr,
		time.Now().Format("2006-01-02 15:04:05"))

	h.sendTextReply(data, result)

	// 保存AI响应到数据库
	if userLog != nil && aiResponseStr != "" {
		_, err := h.chatLogService.CreateAIMessage(username, source, aiResponseStr, userLog.ID)
		if err != nil {
			logx.Error("Failed to save AI response to database: %v", err)
		}
	}

	logx.Info("LLM conversation completed with text")
}
