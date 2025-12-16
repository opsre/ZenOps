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

		// 创建 MCP 服务器 (钉钉和飞书共享)
		mcpServer := imcp.NewMCPServer(cfg)

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
				// 创建 MCP 服务器
				mcpServer := imcp.NewMCPServer(cfg)

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
			return err
		}

		time.Sleep(2 * time.Second)
		logx.Info("👋 Graceful Shutdown Complete.")

		return nil
	},
}
