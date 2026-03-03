package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeroclaw-labs/goclaw/pkg/agent"
	"github.com/zeroclaw-labs/goclaw/pkg/channels"
	"github.com/zeroclaw-labs/goclaw/pkg/config"
	"github.com/zeroclaw-labs/goclaw/pkg/gateway"
	"github.com/zeroclaw-labs/goclaw/pkg/memory"
	"github.com/zeroclaw-labs/goclaw/pkg/providers"
	"github.com/zeroclaw-labs/goclaw/pkg/tools"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "goclaw",
	Short: "GoClaw - Zero overhead autonomous agent runtime",
	Long: `GoClaw - Zero开销，零妥协，100% Go

A high-performance autonomous agent runtime written in Go,
compatible with the ZeroClaw Rust implementation.

Available Providers:
  - openai: OpenAI GPT models
  - anthropic: Anthropic Claude models
  - gemini: Google Gemini models
  - glm: Zhipu AI GLM models
  - ollama: Local Ollama models
  - bedrock: AWS Bedrock models
  - openrouter: OpenRouter multi-provider access

Available Channels:
  - telegram: Telegram Bot API
  - discord: Discord Bot API
  - slack: Slack Bot API
  - whatsapp: WhatsApp Business Cloud API
  - matrix: Matrix Client-Server API
  - dingtalk: DingTalk Bot API
  - email: SMTP Email`,
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
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(channelCmd)
	rootCmd.AddCommand(providerCmd)
	rootCmd.AddCommand(memoryCmd)
	rootCmd.AddCommand(versionCmd)

	agentCmd.Flags().StringVarP(&provider, "provider", "P", "openai", "AI provider to use")
	agentCmd.Flags().StringVarP(&model, "model", "m", "", "Model to use (provider-specific)")
	agentCmd.Flags().Float64Var(&temperature, "temperature", 0.7, "Temperature for generation")
	agentCmd.Flags().BoolVarP(&daemonize, "daemon", "d", false, "Run as daemon")

	gatewayCmd.Flags().StringVarP(&port, "port", "p", "4096", "Port to listen on")
	gatewayCmd.Flags().StringVarP(&provider, "provider", "P", "openai", "AI provider to use")

	daemonCmd.Flags().StringVarP(&daemonPort, "port", "p", "4096", "Port for gateway (if enabled)")

	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelTestCmd)

	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerTestCmd)

	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryClearCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Start the AI agent loop",
	Long:  "Start the GoClaw agent with the specified provider and model",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		providerInstance, err := createProvider(cfg, provider)
		if err != nil {
			return fmt.Errorf("failed to create provider: %w", err)
		}

		// Create memory backend
	memoryBackend := memory.NewNoneMemoryBackend()

	// Create memory implementation
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
		// Email tool
		tools.NewEmailTool("/Users/haha/.zeroclaw/workspace/skills"),
		// Stock analyzer tool
		tools.NewStockAnalyzerTool("/Users/haha/.zeroclaw/workspace/skills"),
		// Memory tools
		tools.NewMemoryStoreTool(memoryBackend),
		tools.NewMemoryRecallTool(memoryBackend),
		tools.NewMemoryForgetTool(memoryBackend),
	}

		// Add IPC tools if enabled
		if ipcDb := tools.GetIpcDb(); ipcDb != nil {
			agentTools = append(agentTools, tools.CreateIpcTools(ipcDb, nil)...)
		}

	

	agt, err := agent.NewAgentBuilder().
	WithProvider(providerInstance).
	WithModelName(model).
	WithTemperature(temperature).
	WithTools(agentTools).
	WithMemory(memImpl).
	Build()
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

		if verbose {
			fmt.Printf("Starting GoClaw Agent with provider: %s, model: %s\n", provider, model)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		if daemonize {
			if verbose {
				fmt.Println("Running in daemon mode...")
			}

			select {
			case <-sigChan:
				fmt.Println("\nReceived shutdown signal")
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			fmt.Println("GoClaw Agent started. Type 'exit' to quit.")

			for {
				select {
				case <-sigChan:
					fmt.Println("\nShutting down...")
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
						fmt.Printf("Error: %v\n", err)
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
	Short: "Start the gateway server",
	Long:  "Start the GoClaw HTTP/WebSocket gateway server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Use config provider if not specified via flag
		providerToUse := provider
		if providerToUse == "openai" && cfg.Provider.Name != "" {
			providerToUse = cfg.Provider.Name
		}

		providerInstance, err := createProvider(cfg, providerToUse)
		if err != nil {
			return fmt.Errorf("failed to create provider: %w", err)
		}

		// Use config model
		modelToUse := cfg.Provider.Model
		if modelToUse == "" {
			modelToUse = "gpt-4"
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
		// Email tool
		tools.NewEmailTool("/Users/haha/.zeroclaw/workspace/skills"),
		// Stock analyzer tool
		tools.NewStockAnalyzerTool("/Users/haha/.zeroclaw/workspace/skills"),
	}

		// Add IPC tools if enabled
		if ipcDb := tools.GetIpcDb(); ipcDb != nil {
			agentTools = append(agentTools, tools.CreateIpcTools(ipcDb, nil)...)
		}

		agt, err := agent.NewAgentBuilder().
			WithProvider(providerInstance).
			WithModelName(modelToUse).
			WithMemory(agent.NewNoneMemoryBackend()).
			WithTools(agentTools).
			Build()
		if err != nil {
			return fmt.Errorf("failed to build agent: %w", err)
		}

		// Create server address
		addr := ":" + daemonPort
	srv := gateway.NewServer(addr, agt)

		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("failed to start gateway: %w", err)
		}

		fmt.Printf("Gateway server started on %s\n", addr)
		fmt.Printf("Provider: %s, Model: %s\n", providerToUse, modelToUse)
		fmt.Printf("HTTP API: http://localhost%s\n", addr)
		fmt.Printf("WebSocket: ws://localhost%s/ws\n", addr)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan
		fmt.Println("\nShutting down gateway...")

		if err := srv.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop gateway: %w", err)
		}

		fmt.Println("Gateway stopped")
		return nil
	},
}
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start long-running autonomous runtime",
	Long:  "Start GoClaw as a daemon with agent and gateway",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Use config provider if not specified via flag
		providerToUse := provider
		if providerToUse == "openai" && cfg.Provider.Name != "" {
			providerToUse = cfg.Provider.Name
		}

		providerInstance, err := createProvider(cfg, providerToUse)
		if err != nil {
			return fmt.Errorf("failed to create provider: %w", err)
		}

		// Use config model
		modelToUse := cfg.Provider.Model
		if modelToUse == "" {
			modelToUse = "gpt-4"
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
		// Email tool
		tools.NewEmailTool("/Users/haha/.zeroclaw/workspace/skills"),
		// Stock analyzer tool
		tools.NewStockAnalyzerTool("/Users/haha/.zeroclaw/workspace/skills"),
	}

		// Add IPC tools if enabled
		if ipcDb := tools.GetIpcDb(); ipcDb != nil {
			agentTools = append(agentTools, tools.CreateIpcTools(ipcDb, nil)...)
		}

		agt, err := agent.NewAgentBuilder().
			WithProvider(providerInstance).
			WithModelName(modelToUse).
			WithMemory(agent.NewNoneMemoryBackend()).
			WithTools(agentTools).
			Build()
		if err != nil {
			return fmt.Errorf("failed to build agent: %w", err)
		}

		addr := ":" + daemonPort
	srv := gateway.NewServer(addr, agt)

		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("failed to start gateway: %w", err)
		}

		fmt.Printf("GoClaw Daemon started on %s\n", addr)
	fmt.Printf("Provider: %s, Model: %s\n", providerToUse, modelToUse)
	fmt.Printf("Available at: http://localhost%s/\n", addr)

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
				log.Printf("Starting channel: %s", name)
				if err := c.Listen(ctx, msgChan); err != nil {
					log.Printf("Channel %s error: %v", name, err)
				}
			}(channelName, ch)
			startedChannels++
			fmt.Printf("  Channel: %s\n", channelName)
		}
		
		if startedChannels == 0 {
			fmt.Println("  No channels configured")
		}
		
		// Start message processor with channel map for replies
		channelWG.Add(1)
		go func() {
			defer channelWG.Done()
			processChannelMessages(ctx, agt, msgChan, channelMap)
		}()

		sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down daemon...")
		
		// Cancel context to stop all goroutines
		cancel()
		
		if err := srv.Stop(context.Background()); err != nil {
			log.Printf("Failed to stop gateway: %v", err)
		}
		
		close(msgChan)
		channelWG.Wait()

		fmt.Println("Daemon stopped")
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
			log.Printf("DingTalk missing client_id or client_secret")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewDingTalkChannel(clientID, clientSecret, allowedUsers)
	case "telegram":
		token := cfg["token"]
		if token == "" {
			log.Printf("Telegram missing token")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewTelegramChannel(token, allowedUsers, false)
	case "discord":
		token := cfg["token"]
		guildID := cfg["guild_id"]
		if token == "" {
			log.Printf("Discord missing token")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewDiscordChannel(token, guildID, allowedUsers, false, false)
	case "slack":
		botToken := cfg["bot_token"]
		signingSecret := cfg["signing_secret"]
		appToken := cfg["app_token"]
		if botToken == "" {
			log.Printf("Slack missing bot_token")
			return nil
		}
		allowedUsers := parseAllowedUsers(cfg["allowed_users"])
		return channels.NewSlackChannel(botToken, signingSecret, appToken, allowedUsers, false)
	default:
		log.Printf("Unknown channel type: %s", name)
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
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}

			resp, err := agt.ProcessMessage(ctx, msg.Content)
			if err != nil {
				log.Printf("Agent error: %v", err)
				continue
			}

			responseText := resp.TextOrEmpty()
			log.Printf("Agent response: %s", responseText)

			ch, ok := channelMap[msg.Channel]
			if !ok {
				log.Printf("No channel found for %s", msg.Channel)
				continue
			}

			replyMsg := types.NewSendMessage(responseText, msg.ReplyTarget)
			if err := ch.Send(ctx, replyMsg); err != nil {
				log.Printf("Failed to send reply: %v", err)
			} else {
				log.Printf("Reply sent successfully to %s", msg.ReplyTarget)
			}
		}
	}
}

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage communication channels",
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available channels",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available channels:")
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
	Short: "Test a channel connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelType := args[0]
		fmt.Printf("Testing channel: %s\n", channelType)

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
			return fmt.Errorf("unknown channel type: %s", channelType)
		}

		if err := ch.HealthCheck(ctx); err != nil {
			return fmt.Errorf("channel health check failed: %w", err)
		}

		fmt.Printf("✓ Channel '%s' is healthy\n", channelType)
		return nil
	},
}

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage AI providers",
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available providers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available providers:")
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
	Short: "Test a provider connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerType := args[0]
		fmt.Printf("Testing provider: %s\n", providerType)

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

		fmt.Printf("✓ Provider '%s' created\n", providerType)
		fmt.Printf("  Capabilities: NativeToolCalling=%v, Vision=%v\n",
			caps.NativeToolCalling,
			caps.Vision,
		)

		return nil
	},
}

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage memory storage",
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List memory entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Memory entries:")
		fmt.Println("(No memory entries yet)")
		return nil
	},
}

var memoryClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all memory entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Memory cleared")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("GoClaw v0.1.0")
		fmt.Println("Built with Go")
		fmt.Println("Compatible with ZeroClaw")
	},
}

func loadConfig() (*config.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.Default(), nil
	}
	configDir := filepath.Join(homeDir, ".zeroclaw")
	return config.Load(configDir)
}

func createProvider(cfg *config.Config, providerType string) (providers.Provider, error) {
	// Check for custom provider URL (format: custom:https://...)
	if strings.HasPrefix(providerType, "custom:") {
		baseURL := strings.TrimPrefix(providerType, "custom:")
		apiKey := cfg.Provider.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		return providers.NewCustomProvider(baseURL, apiKey), nil
	}

	// Get API key from config or environment
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
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerType)
	}
}