package main

import (
	"embed"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeroclaw-labs/goclaw/pkg/agent"
	"github.com/zeroclaw-labs/goclaw/pkg/auth"
	"github.com/zeroclaw-labs/goclaw/pkg/channels"
	"github.com/zeroclaw-labs/goclaw/pkg/config"
	"github.com/zeroclaw-labs/goclaw/pkg/gateway"
	"github.com/zeroclaw-labs/goclaw/pkg/memory"
	"github.com/zeroclaw-labs/goclaw/pkg/onboard"
	"github.com/zeroclaw-labs/goclaw/pkg/providers"
	"github.com/zeroclaw-labs/goclaw/pkg/security"
	"github.com/zeroclaw-labs/goclaw/pkg/skills"
	"github.com/zeroclaw-labs/goclaw/pkg/tools"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

//go:embed web/dist
var embeddedWebFS embed.FS

// memoryImpl implements the agent.Memory interface
type memoryImpl struct {
	backend memory.MemoryBackend
}

func (m *memoryImpl) Recall(ctx context.Context, query string, limit int, category *string) ([]agent.MemoryEntry, error) {
	entries, err := m.backend.Recall(ctx, query, limit, category)
	if err != nil {
		return nil, err
	}

	var result []agent.MemoryEntry
	for _, entry := range entries {
		result = append(result, agent.MemoryEntry{
			Key:       entry.Key,
			Content:   entry.Content,
			Category:  entry.Category,
			Metadata:  entry.Metadata,
		})
	}

	return result, nil
}

func (m *memoryImpl) Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error {
	return m.backend.Store(ctx, key, content, category, metadata)
}

func (m *memoryImpl) Get(ctx context.Context, key string) (*agent.MemoryEntry, error) {
	entry, err := m.backend.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return &agent.MemoryEntry{
		Key:       entry.Key,
		Content:   entry.Content,
		Category:  entry.Category,
		Metadata:  entry.Metadata,
	}, nil
}

func (m *memoryImpl) Search(ctx context.Context, query string, limit int) ([]agent.MemoryEntry, error) {
	entries, err := m.backend.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	var result []agent.MemoryEntry
	for _, entry := range entries {
		result = append(result, agent.MemoryEntry{
			Key:       entry.Key,
			Content:   entry.Content,
			Category:  entry.Category,
			Metadata:  entry.Metadata,
		})
	}

	return result, nil
}

func (m *memoryImpl) Forget(ctx context.Context, key string) error {
	return m.backend.Forget(ctx, key)
}

func (m *memoryImpl) Clear(ctx context.Context) error {
	return m.backend.Clear(ctx)
}

func (m *memoryImpl) List(ctx context.Context, category *string) ([]map[string]interface{}, error) {
	entries, err := m.backend.List(ctx, category)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, entry := range entries {
		result = append(result, map[string]interface{}{
			"id":         entry.ID,
			"key":        entry.Key,
			"content":    entry.Content,
			"category":   entry.Category,
			"metadata":   entry.Metadata,
			"created_at": entry.CreatedAt,
			"updated_at": entry.UpdatedAt,
		})
	}

	return result, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "goclaw",
	Short: "GoClaw - 零开销，零妥协，100% Go",
	Long: `GoClaw - 零开销，零妥协，100% Go

一个用 Go 编写的高性能自主代理运行时，
与 ZeroClaw Rust 实现兼容。

可用的 AI 提供商：
  - openai: OpenAI GPT 模型
  - anthropic: Anthropic Claude 模型
  - gemini: Google Gemini 模型
  - glm: 智谱 AI GLM 模型
  - ollama: 本地 Ollama 模型
  - bedrock: AWS Bedrock 模型
  - openrouter: OpenRouter 多提供商访问
  - gitee: GiteeAI 免费模型
  - bailian: 阿里云百炼

可用的消息通道：
  - telegram: Telegram Bot API
  - discord: Discord Bot API
  - slack: Slack Bot API
  - whatsapp: WhatsApp Business Cloud API
  - matrix: Matrix 客户端-服务端 API
  - dingtalk: 钉钉 Bot API
  - email: SMTP 邮件
  - lark: 飞书 Bot API`,
}

var (
	configPath    string
	verbose       bool
	daemonize     bool
	port          string
	provider      string
	model         string
	temperature   float64
	daemonPort    string
)

func init() {
	rootCmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{.}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)
	rootCmd.SetUsageTemplate(`使用方法:
  {{.UseLine}}

{{if .HasAvailableSubCommands}}
可用命令:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{rpad .Name .NamePadding }} {{if eq .Name "help"}}显示帮助信息{{else}}{{.Short}}{{end}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
标志:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}
全局标志:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}
示例:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}
使用 "{{.CommandPath}} [command] --help" 查看关于某个命令的更多信息。
{{end}}`)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.CompletionOptions.DisableDescriptions = false

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")

	rootCmd.AddCommand(onboardCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(channelCmd)
	rootCmd.AddCommand(providerCmd)
	rootCmd.AddCommand(memoryCmd)
	rootCmd.AddCommand(versionCmd)

	agentCmd.Flags().StringVarP(&provider, "provider", "P", "openai", "AI 提供商")
	agentCmd.Flags().StringVarP(&model, "model", "m", "", "模型名称 (根据提供商而定)")
	agentCmd.Flags().Float64Var(&temperature, "temperature", 0.7, "生成温度")
	agentCmd.Flags().BoolVarP(&daemonize, "daemon", "d", false, "以守护进程方式运行")

	gatewayCmd.Flags().StringVarP(&port, "port", "p", "4096", "监听端口")
	gatewayCmd.Flags().StringVarP(&provider, "provider", "P", "openai", "AI 提供商")

	daemonCmd.Flags().StringVarP(&daemonPort, "port", "p", "4096", "网关端口 (如果启用)")

	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelTestCmd)

	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerTestCmd)

	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryClearCmd)

	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "help" {
			cmd.Short = "显示帮助信息"
		}
	}
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "启动 AI 代理循环",
	Long:  "使用指定的提供商和模型启动 GoClaw 代理",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("配置加载失败: %w", err)
		}

		providerInstance, err := createProvider(cfg, provider)
		if err != nil {
			return fmt.Errorf("创建提供商失败: %w", err)
		}

		memoryBackend := createMemoryBackend(cfg)

		memImpl := &memoryImpl{
			backend: memoryBackend,
		}

	// Create tools for the agent
	agentTools := []tools.Tool{
		// Core tools
		tools.NewShellTool(),
		tools.NewFileReadTool(),
		tools.NewFileWriteTool(),
		tools.NewFileEditTool(),
		tools.NewGlobSearchTool("."),
		tools.NewContentSearchTool("."),
		// Network tools
		tools.NewHTTPTool(),
		tools.NewFetchTool(),
		tools.NewWebFetchTool(),
		tools.NewWebSearchTool(),
		// Image tools
		tools.NewImageInfoTool("."),
		tools.NewScreenshotTool("."),
		// PDF tool
		tools.NewPDFReadTool("."),
		// Schedule tools
		tools.NewScheduleTool("."),
		tools.NewTaskPlanTool(),
		// Cron tools
		tools.NewCronAddTool("."),
		tools.NewCronListTool("."),
		tools.NewCronRemoveTool("."),
		tools.NewCronRunTool("."),
		// Browser tools
		tools.NewBrowserOpenTool(nil),
		tools.NewBrowserTool(nil),
		// Config tools
		tools.NewModelRoutingConfigTool(),
		tools.NewProxyConfigTool(),
		// Delegate tool
		tools.NewDelegateTool("."),
		// Git & Patch tools
		tools.NewApplyPatchTool(),
		tools.NewGitOperationsTool("."),
		// Pushover notification
		tools.NewPushoverTool(""),
		// Memory tools
		tools.NewMemoryStoreTool(memoryBackend),
		tools.NewMemoryRecallTool(memoryBackend),
		tools.NewMemoryForgetTool(memoryBackend),
	}

		// Add IPC tools if enabled
		if ipcDb := tools.GetIpcDb(); ipcDb != nil {
			agentTools = append(agentTools, tools.CreateIpcTools(ipcDb, nil)...)
		}

		// Get skills directory from config
		skillsDir := cfg.GetSkillsDir()

		// Add skill-based tools
		agentTools = append(agentTools,
			//tools.NewEmailTool(skillsDir),
			//tools.NewStockAnalyzerTool(skillsDir),
		)

		promptBuilder := agent.NewDefaultSystemPromptBuilder().WithLocale(cfg.Gateway.Locale)

	agt, err := agent.NewAgentBuilder().
	WithProvider(providerInstance).
	WithModelName(model).
	WithTemperature(temperature).
	WithTools(agentTools).
			WithSkillLoader(skills.NewSkillLoader(skillsDir)).
	WithMemory(memImpl).
	WithAutoSave(cfg.Memory.AutoSave).
	WithPromptBuilder(promptBuilder).
	WithConfig(agent.AgentConfig{
			MaxToolIterations: cfg.Agent.MaxToolIterations,
		}).
	Build()
		if err != nil {
			return fmt.Errorf("创建代理失败: %w", err)
		}

		if verbose {
			fmt.Printf("启动 GoClaw 代理，提供商: %s, 模型: %s\n", provider, model)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		if daemonize {
			if verbose {
				fmt.Println("以守护进程模式运行...")
			}

			select {
			case <-sigChan:
				fmt.Println("\n收到关闭信号")
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			fmt.Println("GoClaw 代理已启动。输入 'exit' 退出。")

			for {
				select {
				case <-sigChan:
					fmt.Println("\n正在关闭...")
					return nil
				default:
					fmt.Print("> ")
					var input string
					fmt.Scanln(&input)

					if input == "exit" {
						return nil
					}

					response, err := agt.ProcessMessage(ctx, input)
					if err != nil {
						fmt.Printf("错误: %v\n", err)
						continue
					}

					fmt.Printf("\n%s\n\n", response.TextOrEmpty())
				}
			}
		}
	},
}

	var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "启动网关服务器",
	Long:  "启动 GoClaw HTTP/WebSocket 网关服务器",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}

		// Use config provider if not specified via flag
		providerToUse := provider
		if providerToUse == "openai" && cfg.Provider.Name != "" {
			providerToUse = cfg.Provider.Name
		}

		providerInstance, err := createProvider(cfg, providerToUse)
		if err != nil {
			return fmt.Errorf("创建提供程序失败: %w", err)
		}

		modelToUse := cfg.Provider.Model
		if modelToUse == "" {
			modelToUse = "gpt-4"
		}

		memoryBackend := createMemoryBackend(cfg)

		agentTools := []tools.Tool{
			tools.NewShellTool(),
			tools.NewFileReadTool(),
			tools.NewFileWriteTool(),
			tools.NewFileEditTool(),
			tools.NewGlobSearchTool("."),
			tools.NewContentSearchTool("."),
			tools.NewHTTPTool(),
			tools.NewFetchTool(),
			tools.NewWebFetchTool(),
			tools.NewWebSearchTool(),
			tools.NewImageInfoTool("."),
			tools.NewScreenshotTool("."),
			tools.NewPDFReadTool("."),
			tools.NewScheduleTool("."),
			tools.NewTaskPlanTool(),
			tools.NewCronAddTool("."),
			tools.NewCronListTool("."),
			tools.NewCronRemoveTool("."),
			tools.NewCronRunTool("."),
			tools.NewBrowserOpenTool(nil),
			tools.NewBrowserTool(nil),
			tools.NewModelRoutingConfigTool(),
			tools.NewProxyConfigTool(),
			tools.NewDelegateTool("."),
			tools.NewApplyPatchTool(),
			tools.NewGitOperationsTool("."),
			tools.NewPushoverTool(""),
			tools.NewMemoryStoreTool(memoryBackend),
			tools.NewMemoryRecallTool(memoryBackend),
			tools.NewMemoryForgetTool(memoryBackend),
		}

		if ipcDb := tools.GetIpcDb(); ipcDb != nil {
			agentTools = append(agentTools, tools.CreateIpcTools(ipcDb, nil)...)
		}

		skillsDir := cfg.GetSkillsDir()

		promptBuilder := agent.NewDefaultSystemPromptBuilder().WithLocale(cfg.Gateway.Locale)

		memImpl := &memoryImpl{
			backend: memoryBackend,
		}

		agt, err := agent.NewAgentBuilder().
			WithProvider(providerInstance).
			WithModelName(modelToUse).
			WithMemory(memImpl).
			WithTools(agentTools).
			WithSkillLoader(skills.NewSkillLoader(skillsDir)).
			WithAutoSave(cfg.Memory.AutoSave).
			WithPromptBuilder(promptBuilder).
			WithConfig(agent.AgentConfig{
				MaxToolIterations: cfg.Agent.MaxToolIterations,
			}).
			Build()
		if err != nil {
			return fmt.Errorf("创建代理失败: %w", err)
		}



		// Create server address
		addr := ":" + daemonPort
		
		// Use embedded web files
		srv := gateway.NewServerWithFS(addr, agt, http.FS(embeddedWebFS))
		
		// Set gateway config from actual values
		srv.SetConfig("provider", providerToUse)
		srv.SetConfig("model", modelToUse)
		srv.SetConfig("temperature", cfg.Agent.Temperature)
		srv.SetConfig("memory_backend", cfg.Memory.Backend)
		srv.SetConfig("wechat_enabled", cfg.Auth.EnableWechatLogin)
		srv.SetConfig("require_pairing", cfg.Gateway.RequirePairing)
		srv.SetConfig("paired_tokens", cfg.Gateway.PairedTokens)
		srv.SetMemoryBackend(memImpl)

		// 创建配对守卫
		if cfg.Gateway.RequirePairing {
			pairingGuard := security.NewPairingGuard(true, []string{"*"})
			srv.SetPairingGuard(pairingGuard)
			log.Printf("配对码: %s", pairingGuard.PairingCode())
			fmt.Printf("配对码: 【  %s  】\n", pairingGuard.PairingCode())
		}

		// 初始化认证服务
		log.Printf("开始初始化认证服务...")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("获取用户主目录失败: %v", err)
		} else {
			log.Printf("用户主目录: %s", homeDir)
			dbPath := filepath.Join(homeDir, ".goclaw", "auth.db")
			log.Printf("认证数据库路径: %s", dbPath)
			userManager, err := auth.NewUserManager(dbPath)
			if err != nil {
				log.Printf("创建用户管理器失败: %v", err)
			} else {
				log.Printf("用户管理器创建成功")
				authService := auth.NewAuthService(
					userManager,
					cfg.Auth.WechatAppID,
					cfg.Auth.WechatAppSecret,
					cfg.Auth.WechatCallbackURL,
				)
				log.Printf("认证服务创建成功")
				srv.SetAuthService(authService)
				srv.SetUserManager(userManager)
				log.Printf("认证服务设置完成")
			}
		}

		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("启动网关失败: %w", err)
		}

		fmt.Printf("网关服务器已启动于 %s\n", addr)
		fmt.Printf("提供程序: %s, 模型: %s\n", providerToUse, modelToUse)
		fmt.Printf("HTTP API: http://localhost%s\n", addr)
		fmt.Printf("WebSocket: ws://localhost%s/ws\n", addr)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan
		fmt.Println("\n正在关闭网关...")

		if err := srv.Stop(ctx); err != nil {
			return fmt.Errorf("关闭网关失败: %w", err)
		}

		fmt.Println("网关已停止")	
		return nil
	},
}
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "启动长期运行的自主运行时",
	Long:  "以守护进程方式启动 GoClaw，包含代理和网关",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}

		// Use config provider if not specified via flag
		providerToUse := provider
		if providerToUse == "openai" && cfg.Provider.Name != "" {
			providerToUse = cfg.Provider.Name
		}

		providerInstance, err := createProvider(cfg, providerToUse)
		if err != nil {
			return fmt.Errorf("创建提供程序失败: %w", err)
		}

		modelToUse := cfg.Provider.Model
		if modelToUse == "" {
			modelToUse = "gpt-4"
		}

		memoryBackend := createMemoryBackend(cfg)

		agentTools := []tools.Tool{
			tools.NewShellTool(),
			tools.NewFileReadTool(),
			tools.NewFileWriteTool(),
			tools.NewFileEditTool(),
			tools.NewGlobSearchTool("."),
			tools.NewContentSearchTool("."),
			tools.NewHTTPTool(),
			tools.NewFetchTool(),
			tools.NewWebFetchTool(),
			tools.NewWebSearchTool(),
			tools.NewImageInfoTool("."),
			tools.NewScreenshotTool("."),
			tools.NewPDFReadTool("."),
			tools.NewScheduleTool("."),
			tools.NewTaskPlanTool(),
			tools.NewCronAddTool("."),
			tools.NewCronListTool("."),
			tools.NewCronRemoveTool("."),
			tools.NewCronRunTool("."),
			tools.NewBrowserOpenTool(nil),
			tools.NewBrowserTool(nil),
			tools.NewModelRoutingConfigTool(),
			tools.NewProxyConfigTool(),
			tools.NewDelegateTool("."),
			tools.NewApplyPatchTool(),
			tools.NewGitOperationsTool("."),
			tools.NewPushoverTool(""),
			tools.NewMemoryStoreTool(memoryBackend),
			tools.NewMemoryRecallTool(memoryBackend),
			tools.NewMemoryForgetTool(memoryBackend),
		}

		if ipcDb := tools.GetIpcDb(); ipcDb != nil {
			agentTools = append(agentTools, tools.CreateIpcTools(ipcDb, nil)...)
		}

		skillsDir := cfg.GetSkillsDir()
		log.Printf("技能目录: %s", skillsDir)

		promptBuilder := agent.NewDefaultSystemPromptBuilder().WithLocale(cfg.Gateway.Locale)

		skillLoader := skills.NewSkillLoader(skillsDir)
		log.Printf("技能加载器创建成功: %v", skillLoader)

		memImpl := &memoryImpl{
			backend: memoryBackend,
		}

		log.Printf("构建代理...")
		agt, err := agent.NewAgentBuilder().
			WithProvider(providerInstance).
			WithModelName(modelToUse).
			WithMemory(memImpl).
			WithTools(agentTools).
			WithSkillLoader(skillLoader).
			WithAutoSave(cfg.Memory.AutoSave).
			WithPromptBuilder(promptBuilder).
			WithConfig(agent.AgentConfig{
				MaxToolIterations: cfg.Agent.MaxToolIterations,
			}).
			Build()
		if err != nil {
			log.Printf("构建代理失败: %v", err)
			return fmt.Errorf("构建代理失败: %w", err)
		}
		log.Printf("代理构建成功")

		addr := ":" + daemonPort
		log.Printf("创建网关服务器，地址: %s", addr)
		
		// Use embedded web files
		srv := gateway.NewServerWithFS(addr, agt, http.FS(embeddedWebFS))
		
		// Set gateway config from actual values
		srv.SetConfig("provider", providerToUse)
		srv.SetConfig("model", modelToUse)
		srv.SetConfig("temperature", cfg.Agent.Temperature)
		srv.SetConfig("memory_backend", cfg.Memory.Backend)
		srv.SetConfig("wechat_enabled", cfg.Auth.EnableWechatLogin)
		srv.SetConfig("require_pairing", cfg.Gateway.RequirePairing)
		srv.SetConfig("paired_tokens", cfg.Gateway.PairedTokens)

		// 创建配对守卫
		if cfg.Gateway.RequirePairing {
			pairingGuard := security.NewPairingGuard(true, []string{"*"})
			srv.SetPairingGuard(pairingGuard)
			log.Printf("配对码: %s", pairingGuard.PairingCode())
			fmt.Printf("配对码: 【  %s  】\n", pairingGuard.PairingCode())
		}

		// 初始化认证服务
		log.Printf("开始初始化认证服务...")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("获取用户主目录失败: %v", err)
		} else {
			log.Printf("用户主目录: %s", homeDir)
			dbPath := filepath.Join(homeDir, ".goclaw", "auth.db")
			log.Printf("认证数据库路径: %s", dbPath)
			userManager, err := auth.NewUserManager(dbPath)
			if err != nil {
				log.Printf("创建用户管理器失败: %v", err)
			} else {
				log.Printf("用户管理器创建成功")
				authService := auth.NewAuthService(
					userManager,
					cfg.Auth.WechatAppID,
					cfg.Auth.WechatAppSecret,
					cfg.Auth.WechatCallbackURL,
				)
				log.Printf("认证服务创建成功")
				srv.SetAuthService(authService)
				srv.SetUserManager(userManager)
				log.Printf("认证服务设置完成")
			}
		}

		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("启动网关失败: %w", err)
		}

		fmt.Printf("网关服务器已启动于 %s\n", addr)
	fmt.Printf("提供程序: %s, 模型: %s\n", providerToUse, modelToUse)
	fmt.Printf("HTTP API: http://localhost%s\n", addr)
	fmt.Printf("WebSocket: ws://localhost%s/ws\n", addr)	

		// Start channels from config
		var channelWG sync.WaitGroup
		msgChan := make(chan types.ChannelMessage, 100)
		
		// Create channel map for replies
		channelMap := make(map[string]channels.Channel)
		
		// Start configured channels
		startedChannels := 0
		for channelName, channelCfg := range cfg.Channels {
			ch := createChannelFromConfig(channelName, channelCfg)
			if ch == nil {
				continue
			}
			
			channelMap[channelName] = ch
			
			channelWG.Add(1)
			go func(name string, c channels.Channel) {
				defer channelWG.Done()
				log.Printf("启动通道: %s", name)
				if err := c.Listen(ctx, msgChan); err != nil {
					log.Printf("通道 %s 错误: %v", name, err)
				}
			}(channelName, ch)
			startedChannels++
			fmt.Printf("  通道: %s\n", channelName)
		}
		
		if startedChannels == 0 {
			fmt.Println("  未配置任何通道")
		}
		
		// Start message processor with channel map for replies
	channelWG.Add(1)
	go func() {
		defer channelWG.Done()
		log.Printf("启动消息处理器, channelMap size=%d", len(channelMap))
		for chName := range channelMap {
			log.Printf("  通道: %s", chName)
		}
		processChannelMessages(ctx, agt, msgChan, channelMap)
	}()

		sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		fmt.Println("\n关闭守护进程...")
		log.Printf("收到信号，开始关闭守护进程")
		
		// Cancel context to stop all goroutines
		cancel()
		
		if err := srv.Stop(context.Background()); err != nil {
			log.Printf("关闭网关服务器失败: %v", err)
		}
		
		close(msgChan)
		channelWG.Wait()

		fmt.Println("守护进程已停止")
		os.Exit(0)
	}()

	// Wait for context to be canceled
	<-ctx.Done()
	return nil
	},
}

// createChannelFromConfig creates a channel instance from config
func createChannelFromConfig(name string, cfg config.ChannelConfig) channels.Channel {
	switch name {
	case "dingtalk":
		clientID := cfg["client_id"]
		clientSecret := cfg["client_secret"]
		if clientID == "" || clientSecret == "" {
			log.Printf("DingTalk 通道缺少 client_id 或 client_secret")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewDingTalkChannel(clientID, clientSecret, allowedUsers)
	case "telegram":
		token := cfg["token"]
		if token == "" {
			log.Printf("Telegram 通道缺少 token")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewTelegramChannel(token, allowedUsers, false)
	case "discord":
		token := cfg["token"]
		guildID := cfg["guild_id"]
		if token == "" {
			log.Printf("Discord 通道缺少 token")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewDiscordChannel(token, guildID, allowedUsers, false, false)
	case "slack":
		botToken := cfg["bot_token"]
		signingSecret := cfg["signing_secret"]
		appToken := cfg["app_token"]
		if botToken == "" {
			log.Printf("Slack 通道缺少 bot_token")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewSlackChannel(botToken, signingSecret, appToken, allowedUsers, false)
	case "lark":
		appID := cfg["app_id"]
		appSecret := cfg["app_secret"]
		if appID == "" || appSecret == "" {
			log.Printf("Lark 通道缺少 app_id 或 app_secret")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewLarkChannel(appID, appSecret, allowedUsers)
	default:
		log.Printf("未知通道类型: %s", name)
		return nil
	}
}

// parseAllowedUsers parses allowed_users from config
func parseAllowedUsers(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	s = strings.Trim(s, "[]")
	var users []string
	for _, u := range strings.Split(s, ",") {
		u = strings.TrimSpace(strings.Trim(u, "\"'"))
		if u != "" {
			users = append(users, u)
		}
	}
	return users
}

// processChannelMessages processes incoming channel messages
func processChannelMessages(ctx context.Context, agt *agent.Agent, msgChan <-chan types.ChannelMessage, channelMap map[string]channels.Channel) {
	log.Printf("processChannelMessages: 开始处理消息")
	for {
		select {
		case <-ctx.Done():
			log.Printf("processChannelMessages: context cancelled")
			return
		case msg, ok := <-msgChan:
			if !ok {
				log.Printf("processChannelMessages: msgChan closed")
				return
			}
			log.Printf("processChannelMessages: 收到消息, channel=%s, sender=%s", msg.Channel, msg.Sender)

			log.Printf("=== 处理通道消息 ===")
			log.Printf("  通道: %s, 发送者: %s, 回复目标: %s", msg.Channel, msg.Sender, msg.ReplyTarget)
			log.Printf("  内容: %s", msg.Content)

			// 使用流式卡片显示加载效果
			var streamingSession *channels.LarkStreamingSession
			if ch, ok := channelMap[msg.Channel]; ok {
				if larkCh, ok := ch.(*channels.LarkChannel); ok {
					log.Printf("  创建流式卡片...")
					streamingSession = channels.NewLarkStreamingSession(larkCh.AppID(), larkCh.AppSecret(), "")
					// 使用 message_id 进行回复
					messageID := msg.MessageID
					if messageID == "" {
						messageID = msg.ReplyTarget
					}
					if err := streamingSession.Start(ctx, messageID, "chat_id", msg.Content); err != nil {
						log.Printf("  创建流式卡片失败: %v", err)
						streamingSession = nil
					} else {
						log.Printf("  流式卡片已创建")
					}
				}
			}

			resp, err := agt.ProcessMessage(ctx, msg.Content)
			if err != nil {
				log.Printf("代理错误: %v", err)
				if streamingSession != nil {
					streamingSession.Close(ctx, "处理消息时出错")
				}
				continue
			}

			responseText := resp.TextOrEmpty()
			log.Printf("  响应长度: %d 个字符", len(responseText))
			log.Printf("  响应预览: %s", truncateString(responseText, 100))

			ch, ok := channelMap[msg.Channel]
			if !ok {
				log.Printf("未找到通道 %s", msg.Channel)
				if streamingSession != nil {
					streamingSession.Close(ctx, "")
				}
				continue
			}

			// 更新流式卡片
			if streamingSession != nil {
				log.Printf("  更新流式卡片内容...")
				if err := streamingSession.Update(ctx, responseText); err != nil {
					log.Printf("  更新流式卡片失败: %v", err)
				}
				log.Printf("  关闭流式卡片...")
				if err := streamingSession.Close(ctx, responseText); err != nil {
					log.Printf("  关闭流式卡片失败: %v", err)
				}
			}

			replyMsg := types.NewSendMessage(responseText, msg.ReplyTarget)
			log.Printf("  发送回复到 %s...", msg.ReplyTarget)
			if err := ch.Send(ctx, replyMsg); err != nil {
				log.Printf("发送回复失败: %v", err)
			} else {
				log.Printf("  回复发送成功!")
			}
			log.Printf("=== 处理通道消息结束 ===")
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "管理消息通道",
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用的消息通道",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("可用通道:")
		fmt.Println("  - telegram:  Telegram Bot API")
		fmt.Println("  - discord:   Discord Bot API")
		fmt.Println("  - slack:     Slack Bot API")
		fmt.Println("  - whatsapp:  WhatsApp Business Cloud API")
		fmt.Println("  - matrix:    Matrix Client-Server API")
		fmt.Println("  - dingtalk:  DingTalk Bot API")
		fmt.Println("  - email:     SMTP Email")
	},
}

var channelTestCmd = &cobra.Command{
	Use:   "test <channel>",
	Short: "测试通道连接",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelType := args[0]
		fmt.Printf("测试通道: %s\n", channelType)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var ch channels.Channel

		switch channelType {
		case "telegram":
			ch = channels.NewTelegramChannel("test-token", []string{}, false)
		case "discord":
			ch = channels.NewDiscordChannel("test-token", "", []string{}, false, false)
		case "slack":
			ch = channels.NewSlackChannel("test-token", "", "", []string{}, false)
		case "whatsapp":
			ch = channels.NewWhatsAppChannel("test-token", "123456789", "verify", []string{})
		case "matrix":
			ch = channels.NewMatrixChannel("https://matrix.org", "test-token", "!room:matrix.org", []string{}, false)
		case "dingtalk":
			ch = channels.NewDingTalkChannel("client-id", "client-secret", []string{})
		case "email":
			ch = channels.NewEmailChannel("smtp.gmail.com", 587, "user", "pass", "from@example.com", []string{})
		default:
			return fmt.Errorf("未知通道类型: %s", channelType)
		}

		if err := ch.HealthCheck(ctx); err != nil {
			return fmt.Errorf("通道 '%s' 健康检查失败: %w", channelType, err)
		}

		fmt.Printf("✓ 通道 '%s' 健康\n", channelType)
		return nil
	},
}

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "管理 AI 提供商",
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用的 AI 提供商",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("可用 AI 提供商:")
		fmt.Println("  - openai:     OpenAI GPT models")
		fmt.Println("  - anthropic:  Anthropic Claude models")
		fmt.Println("  - gemini:     Google Gemini models")
		fmt.Println("  - glm:        Zhipu AI GLM models")
		fmt.Println("  - ollama:     Local Ollama models")
		fmt.Println("  - bedrock:    AWS Bedrock models")
		fmt.Println("  - openrouter: OpenRouter multi-provider access")
	},
}

var providerTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "测试提供商连接",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerType := args[0]
		fmt.Printf("测试 AI 提供商: %s\n", providerType)

		var p providers.Provider

		switch providerType {
		case "openai":
			p = providers.NewOpenAIProvider("test-key")
		case "anthropic":
			p = providers.NewAnthropicProvider("test-key")
		case "gemini":
			p = providers.NewGeminiProvider("test-key")
		case "glm":
			p = providers.NewGLMProvider("test-key")
		case "ollama":
			p = providers.NewOllamaProvider()
		case "bedrock":
			p = providers.NewBedrockProvider("test-key", "test-secret", "", "us-east-1")
		case "openrouter":
			p = providers.NewOpenRouterProvider("test-key")
		default:
			return fmt.Errorf("unknown provider type: %s", providerType)
		}

caps := p.Capabilities()

		fmt.Printf("✓ AI 提供商 '%s' 创建成功\n", providerType)
		fmt.Printf("  功能: NativeToolCalling=%v, Vision=%v\n",
			caps.NativeToolCalling,
			caps.Vision,
		)

		return nil
	},
}

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "管理存储",
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出存储条目",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("存储条目:")
		fmt.Println("(暂无存储条目)")
		return nil
	},
}

var memoryClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清除所有存储条目",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("所有存储条目已清除")	
		return nil
	},
}

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "交互式配置向导",
	Long:  "运行交互式配置向导来配置 GoClaw",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		wizard := onboard.NewWizard()
		return wizard.Run(ctx)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("GoClaw v0.1.0")
		fmt.Println("构建于 Go")
		fmt.Println("与 ZeroClaw 兼容")
	},
}

func loadConfig() (*config.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.Default(), nil
	}
	configDir := filepath.Join(homeDir, ".goclaw")
	
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := initializeConfig(configDir); err != nil {
			log.Printf("警告: 初始化配置失败: %v", err)
			return config.Default(), nil
		}
	}
	
	return config.Load(configDir)
}

func createMemoryBackend(cfg *config.Config) memory.MemoryBackend {
	backendType := cfg.Memory.Backend
	if backendType == "" {
		backendType = "none"
	}

	backend, err := memory.NewBackend(backendType, cfg.Memory.Config)
	if err != nil {
		log.Printf("警告: 创建内存后端失败 (%s): %v, 使用none后端", backendType, err)
		return memory.NewNoneMemoryBackend()
	}

	log.Printf("使用内存后端: %s", backendType)
	return backend
}

func initializeConfig(configDir string) error {
	log.Printf("初始化 GoClaw 配置在 %s", configDir)	
	
	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	
	// Create workspace directory
	workspaceDir := filepath.Join(configDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("创建工作空间目录失败: %w", err)
	}
	
	// Create skills directory
	skillsDir := filepath.Join(configDir, "workspace", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}
	
	// Create default config file
	configPath := filepath.Join(configDir, "config.toml")
	defaultConfig := `# GoClaw Configuration
# Generated automatically on first run

[provider]
name = "openai"
model = "gpt-4"
# api_key = "your-api-key-here"

[agent]
max_tool_iterations = 15

[memory]
backend = "none"

[gateway]
host = "0.0.0.0"
port = 4096
`
	
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	
	log.Printf("配置初始化成功")
	log.Printf("  配置文件: %s", configPath)
	log.Printf("  工作空间: %s", workspaceDir)
	log.Printf("  技能目录: %s", skillsDir)
	log.Printf("")
	log.Printf("运行 'goclaw onboard' 来配置您的 AI 提供商和设置")
	
	return nil
}

func createProvider(cfg *config.Config, providerType string) (providers.Provider, error) {
	if strings.HasPrefix(providerType, "custom:") {
		baseURL := strings.TrimPrefix(providerType, "custom:")
		apiKey := cfg.Provider.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		return providers.NewCustomProvider(baseURL, apiKey), nil
	}

	apiKey := cfg.Provider.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	switch providerType {
	case "openai":
		return providers.NewOpenAIProvider(apiKey), nil
	case "anthropic":
		return providers.NewAnthropicProvider(apiKey), nil
	case "gemini":
		return providers.NewGeminiProvider(apiKey), nil
	case "glm":
		return providers.NewGLMProvider(apiKey), nil
	case "ollama":
		return providers.NewOllamaProvider(), nil
	case "bedrock":
		return providers.NewBedrockProvider("", "", "", "us-east-1"), nil
	case "openrouter":
		return providers.NewOpenRouterProvider(apiKey), nil
	case "gitee":
		return providers.NewCustomProvider("https://ai.gitee.com/v1", apiKey), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerType)
	}
}