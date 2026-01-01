package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/mcpclient"
	_ "github.com/eryajf/zenops/internal/provider/aliyun"  // 注册 aliyun provider
	_ "github.com/eryajf/zenops/internal/provider/jenkins" // 注册 jenkins provider
	_ "github.com/eryajf/zenops/internal/provider/tencent" // 注册 tencent provider
	"github.com/eryajf/zenops/internal/server"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config

	httpOnly bool
	mcpOnly  bool

	Version   string
	GitCommit string
	BuildTime string
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "zenops",
	Short: "ZenOps - 运维数据智能化查询工具",
	Long: `ZenOps 是一个面向运维领域的数据智能化查询工具,
通过统一的接口抽象,支持多云平台(阿里云、腾讯云等)、CI/CD 工具(Jenkins等)的资源查询,
并通过 CLI、HTTP API 和 MCP 协议提供多种访问方式。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 加载配置
		var err error
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			logx.Fatal("failed to load config: %v", err)
		}
		config.SetGlobalConfig(cfg)

		return nil
	},
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "配置文件路径 (默认: config.yaml)")
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(fmt.Sprintf(`🍉 {{with .Name}}{{printf "%%s version information: " .}}{{end}}
    {{printf "Version:    %%s" .Version}}
    Git Commit: %s
    Go version: %s
    OS/Arch:    %s/%s
    Build Time: %s
`, GitCommit, runtime.Version(), runtime.GOOS, runtime.GOARCH, BuildTime))
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listToolsCmd)

	runCmd.Flags().BoolVar(&httpOnly, "http-only", false, "仅启动 HTTP 服务")
	runCmd.Flags().BoolVar(&mcpOnly, "mcp-only", false, "仅启动 MCP 服务")
}

// runCmd 服务命令
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动 HTTP 或 MCP 服务，同时启动钉钉/飞书Stream服务",
	Long:  `启动 ZenOps 的 HTTP API 服务器或 MCP 协议服务器,或同时启动两者。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logx.Info("🧘 Starting ZenOps Server, Version %s", Version)

		// 初始化版本信息处理器
		server.InitVersionHandler(Version, GitCommit, BuildTime)

		// 检查 flag 冲突
		if httpOnly && mcpOnly {
			return fmt.Errorf("--http-only 和 --mcp-only 不能同时使用")
		}

		// 确定要启动的服务
		startHTTP := !mcpOnly && cfg.Server.HTTP.Enabled
		startMCP := !httpOnly && cfg.Server.MCP.Enabled

		// 如果使用了 --http-only，即使配置文件中 HTTP 未启用也要启动
		if httpOnly {
			startHTTP = true
			startMCP = false
		}

		// 如果使用了 --mcp-only，即使配置文件中 MCP 未启用也要启动
		if mcpOnly {
			startMCP = true
			startHTTP = false
		}

		// 创建 context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 监听退出信号
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// 错误通道
		errCh := make(chan error, 3)

		// 1. 创建 MCP 客户端管理器
		mcpClientManager := mcpclient.NewManager()

		// 设置为全局 MCP 管理器,供 HTTP API 使用
		server.SetGlobalMCPManager(mcpClientManager)

		// 2. 从数据库加载并连接已启用的 MCP 服务器
		logx.Info("🗄️  Loading MCP servers from database...")
		if err := server.InitializeMCPServersFromDB(ctx, mcpClientManager); err != nil {
			logx.Warn("⚠️  Failed to initialize MCP servers from database: %v", err)
		}

		// 3. 加载外部 MCP 配置文件 (兼容旧配置方式)
		if cfg.MCPServersConfig != "" {
			logx.Info("📥 Loading external MCP servers from: %s", cfg.MCPServersConfig)
			mcpServersConfig, err := config.LoadMCPServersConfig(cfg.MCPServersConfig)
			if err != nil {
				logx.Warn("⚠️  Failed to load MCP servers config: %v", err)
			} else {
				// 注册所有外部 MCP 客户端
				if err := mcpClientManager.LoadFromConfig(mcpServersConfig); err != nil {
					logx.Error("❌ Failed to load MCP clients: %v", err)
				}
			}
		}

		// 4. 创建 MCP 服务器 (钉钉和飞书共享)
		mcpServer := imcp.NewMCPServer(cfg)

		// 5. 注册外部 MCP 的工具 (如果启用)
		if cfg.Server.MCP.AutoRegisterExternalTools {
			logx.Info("🔧 Registering external MCP tools...")
			if err := mcpServer.RegisterExternalMCPTools(ctx, mcpClientManager); err != nil {
				logx.Error("❌ Failed to register external MCP tools: %v", err)
			}
		}

		// 启动钉钉服务 (Stream模式)
		if cfg.DingTalk.Enabled {
			go func() {
				// 创建钉钉服务
				dingTalkService, err := server.NewDingTalkService(cfg, mcpServer)
				if err != nil {
					errCh <- fmt.Errorf("failed to create dingtalk service: %w", err)
					return
				}

				// 启动钉钉服务
				if err := dingTalkService.Start(ctx); err != nil {
					errCh <- fmt.Errorf("dingtalk service error: %w", err)
					return
				}
			}()
		}

		// 启动飞书服务 (Stream模式)
		if cfg.Feishu.Enabled {
			go func() {
				// 创建飞书服务
				feishuService, err := server.NewFeishuStreamServer(cfg, mcpServer)
				if err != nil {
					errCh <- fmt.Errorf("failed to create feishu service: %w", err)
					return
				}

				// 启动飞书服务
				if err := feishuService.Start(); err != nil {
					errCh <- fmt.Errorf("feishu service error: %w", err)
					return
				}
			}()
		}

		// 启动 HTTP 服务
		if startHTTP {
			logx.Info("🌐 Starting HTTP server...")
			go func() {
				// 创建 HTTP 服务器 (使用 Gin)
				httpServer := server.NewHTTPGinServer(cfg)

				// 设置 MCP Server (用于企业微信等需要 MCP 的功能)
				httpServer.SetMCPServer(mcpServer)

				// 启动 HTTP 服务器(阻塞式)
				if err := httpServer.Start(); err != nil {
					errCh <- fmt.Errorf("http server error: %w", err)
				}
			}()
		}

		// 启动 MCP 服务
		if startMCP {
			logx.Info("🔌 Starting MCP server...")
			go func() {
				// 使用已经注册了外部工具的 MCP 服务器
				err := mcpServer.StartSSE()
				if err != nil {
					errCh <- fmt.Errorf("mcp server error: %w", err)
				}
			}()
		}

		// 如果没有任何服务启动，给出提示
		if !startHTTP && !startMCP && !cfg.DingTalk.Enabled && !cfg.Feishu.Enabled {
			logx.Warn("⚠️  No services enabled. Please check your configuration or use --http-only or --mcp-only flags.")
		}

		// 等待退出信号或错误
		select {
		case sig := <-sigCh:
			logx.Info("📬 Received Signal, Shutting Down Now, Signal %s", sig.String())
			cancel()
		case err := <-errCh:
			logx.Error("Server error: %v", err)
			cancel()
			// 清理外部 MCP 客户端
			mcpClientManager.CloseAll()
			return err
		}

		// 清理外部 MCP 客户端
		logx.Info("🧹 Cleaning up external MCP clients...")
		mcpClientManager.CloseAll()

		time.Sleep(2 * time.Second)
		logx.Info("👋 Graceful Shutdown Complete.")

		return nil
	},
}

// listToolsCmd 列出所有已注册的工具
var listToolsCmd = &cobra.Command{
	Use:   "list-tools",
	Short: "列出所有已注册的 MCP 工具(包括内置和外部工具)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 1. 创建 MCP 客户端管理器
		mcpClientManager := mcpclient.NewManager()

		// 2. 加载外部 MCP 配置
		if cfg.MCPServersConfig != "" {
			mcpServersConfig, err := config.LoadMCPServersConfig(cfg.MCPServersConfig)
			if err != nil {
				logx.Warn("⚠️  Failed to load MCP servers config: %v", err)
			} else {
				if err := mcpClientManager.LoadFromConfig(mcpServersConfig); err != nil {
					logx.Error("❌ Failed to load MCP clients: %v", err)
				}
			}
		}

		// 3. 创建 MCP 服务器
		mcpServer := imcp.NewMCPServer(cfg)

		// 4. 注册外部 MCP 的工具
		if cfg.Server.MCP.AutoRegisterExternalTools {
			if err := mcpServer.RegisterExternalMCPTools(ctx, mcpClientManager); err != nil {
				logx.Error("❌ Failed to register external MCP tools: %v", err)
			}
		}

		// 5. 列出所有工具
		result, err := mcpServer.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}

		fmt.Printf("\n📋 Total Tools: %d\n\n", len(result.Tools))

		// 分类统计
		internalTools := []string{}
		externalTools := []string{}

		for _, tool := range result.Tools {
			// 根据工具名前缀判断是否为外部工具
			// 外部工具通常有前缀（如 jenkins_, github_ 等）
			if cfg.MCPServersConfig != "" {
				// 简单判断：如果工具名包含下划线且不是内置工具，可能是外部工具
				isInternal := false
				internalToolNames := []string{
					"search_ecs_by_ip", "search_ecs_by_name", "list_ecs",
					"search_rds_by_name", "list_rds", "get_rds_info",
					"list_slb", "get_slb_info", "search_slb_by_ip",
					"list_oss_buckets", "get_oss_bucket_info",
					"list_redis", "get_redis_info",
					"search_eip_by_ip", "list_eip",
					"search_nat_by_ip", "list_nat",
					"list_cvm", "search_cvm_by_ip", "search_cvm_by_name",
				}
				for _, name := range internalToolNames {
					if tool.Name == name {
						isInternal = true
						break
					}
				}

				if isInternal {
					internalTools = append(internalTools, tool.Name)
				} else {
					externalTools = append(externalTools, tool.Name)
				}
			} else {
				internalTools = append(internalTools, tool.Name)
			}
		}

		// 打印内置工具
		if len(internalTools) > 0 {
			fmt.Printf("🔧 Internal Tools (%d):\n", len(internalTools))
			for i, name := range internalTools {
				fmt.Printf("  %d. %s\n", i+1, name)
			}
			fmt.Println()
		}

		// 打印外部工具
		if len(externalTools) > 0 {
			fmt.Printf("🌐 External Tools (%d):\n", len(externalTools))
			for i, name := range externalTools {
				fmt.Printf("  %d. %s\n", i+1, name)
			}
			fmt.Println()
		}

		// 关闭客户端
		mcpClientManager.CloseAll()

		return nil
	},
}
