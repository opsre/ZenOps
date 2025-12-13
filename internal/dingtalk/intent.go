package dingtalk

import (
	"fmt"
	"regexp"
	"strings"

	"cnb.cool/zhiqiangwang/pkg/logx"
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

// NewIntentParser 创建意图解析器
func NewIntentParser() *IntentParser {
	parser := &IntentParser{
		patterns: make([]intentPattern, 0),
	}

	// 注册所有意图模式
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
			// 提取区域信息
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
			logx.Debug("Pattern matched, pattern %s, matches %v",
				pattern.regex.String(),
				matches)

			intent := &Intent{
				Provider: pattern.provider,
				Resource: pattern.resource,
				Action:   pattern.action,
				Params:   pattern.extractor(matches),
			}

			// 映射到 MCP 工具
			intent.MCPTool = p.mapToMCPTool(intent)

			logx.Info("Intent parsed, provider %s, resource %s, action %s, mcp_tool %s, params %v",
				intent.Provider,
				intent.Resource,
				intent.Action,
				intent.MCPTool,
				intent.Params)

			return intent, nil
		}
	}

	return nil, fmt.Errorf("无法识别您的请求,请尝试更明确的描述")
}

// mapToMCPTool 将意图映射到 MCP 工具
func (p *IntentParser) mapToMCPTool(intent *Intent) string {
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

// GetHelpMessage 获取帮助消息
func GetHelpMessage() string {
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
