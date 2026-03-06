package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	Agent       AgentConfig
	Provider    ProviderConfig
	Memory      MemoryConfig
	Gateway     GatewayConfig
	Auth        AuthConfig
	Channels    map[string]ChannelConfig
	ChannelsRaw string // Raw TOML content for channels
	SkillsDir   string // Skills directory path
}

type AgentConfig struct {
	SystemPrompt        string
	MaxTokens           int
	Temperature          float64
	MaxToolIterations    int
}

type ProviderConfig struct {
	Name    string
	APIKey  string
	Model   string
	BaseURL string
}

type MemoryConfig struct {
	Backend   string
	Config    map[string]string
	AutoSave  bool
}

type GatewayConfig struct {
	Host           string
	Port           int
	StaticDir      string // Static files directory for web interface
	RequirePairing bool   // 是否需要配对码
	PairedTokens   []string // 已配对的 token 列表
}

type AuthConfig struct {
	EnableLogin      bool   // 是否启用登录功能
	EnableWechatLogin bool   // 是否启用微信登录
	EnableAudit      bool   // 是否启用管理员审核
	WechatAppID      string // 微信AppID
	WechatAppSecret  string // 微信AppSecret
	WechatCallbackURL string // 微信回调地址
}

type ChannelConfig map[string]string

// DingTalkConfig holds DingTalk channel configuration
type DingTalkConfig struct {
	ClientID     string
	ClientSecret string
	AllowedUsers []string
}

func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultSkillsDir := "."
	if homeDir != "" {
		defaultSkillsDir = filepath.Join(homeDir, ".goclaw", "workspace", "skills")
	}

	// Set default static directory
	defaultStaticDir := "./web/dist"
	if homeDir != "" {
		defaultStaticDir = filepath.Join(homeDir, ".goclaw", "web", "dist")
	}

	return &Config{
		Agent: AgentConfig{
			SystemPrompt:        "You are GoClaw, a helpful AI assistant.",
			MaxTokens:           4096,
			Temperature:          0.7,
			MaxToolIterations:    15,
		},
		Provider: ProviderConfig{
			Name:  "openai",
			Model: "gpt-4",
		},
		Memory: MemoryConfig{
			Backend: "none",
			Config:  make(map[string]string),
		},
		Gateway: GatewayConfig{
			Host:      "0.0.0.0",
			Port:      8080,
			StaticDir: defaultStaticDir,
		},
		Auth: AuthConfig{
			EnableLogin:      false, // 默认禁用登录功能
			EnableWechatLogin: false, // 默认禁用微信登录
			EnableAudit:      false, // 默认禁用管理员审核
		},
		Channels:  make(map[string]ChannelConfig),
		SkillsDir: defaultSkillsDir,
	}
}

func Load(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return Default(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	content := string(data)

	cfg.Provider.Name = parseTomlString(content, "default_provider", cfg.Provider.Name)
	cfg.Provider.Model = parseTomlString(content, "default_model", cfg.Provider.Model)
	cfg.Provider.APIKey = parseTomlString(content, "api_key", "")
	cfg.Provider.BaseURL = parseTomlString(content, "base_url", "")
	cfg.Agent.Temperature = parseTomlFloat(content, "default_temperature", cfg.Agent.Temperature)
	cfg.Agent.MaxToolIterations = parseTomlInt(content, "max_tool_iterations", cfg.Agent.MaxToolIterations)
	cfg.SkillsDir = parseTomlString(content, "skills_dir", cfg.SkillsDir)
	cfg.Gateway.StaticDir = parseTomlString(content, "static_dir", cfg.Gateway.StaticDir)
	cfg.Gateway.Port = parseTomlNestedInt(content, "gateway.port", cfg.Gateway.Port)
	cfg.Gateway.RequirePairing = parseTomlNestedString(content, "gateway.require_pairing", "false") == "true"
	cfg.Gateway.PairedTokens = parseTomlNestedStringArray(content, "gateway.paired_tokens", cfg.Gateway.PairedTokens)

	cfg.Memory.Backend = parseTomlNestedString(content, "memory.backend", cfg.Memory.Backend)
	cfg.Memory.Config = parseMemoryConfig(content)
	cfg.Memory.AutoSave = parseTomlNestedString(content, "memory.auto_save", "false") == "true"

	if strings.HasPrefix(cfg.Provider.Name, "custom:") {
		cfg.Provider.BaseURL = strings.TrimPrefix(cfg.Provider.Name, "custom:")
	}

	// 解析认证配置
	cfg.Auth.EnableLogin = parseTomlNestedString(content, "auth.enable_login", "false") == "true"
	cfg.Auth.EnableWechatLogin = parseTomlNestedString(content, "wechat.enabled", "false") == "true"
	cfg.Auth.EnableAudit = parseTomlNestedString(content, "auth.enable_audit", "false") == "true"
	cfg.Auth.WechatAppID = parseTomlNestedString(content, "wechat.app_id", "")
	cfg.Auth.WechatAppSecret = parseTomlNestedString(content, "wechat.app_secret", "")
	cfg.Auth.WechatCallbackURL = parseTomlNestedString(content, "wechat.redirect_uri", "")

	cfg.Channels = parseChannelsConfig(content)

	return cfg, nil
}

// parseMemoryConfig parses [memory] section
func parseMemoryConfig(content string) map[string]string {
	config := make(map[string]string)

	sectionRe := regexp.MustCompile(`(?m)^\[memory\]`)
	match := sectionRe.FindStringIndex(content)

	if match == nil {
		return config
	}

	start := match[1]
	
	nextSectionRe := regexp.MustCompile(`(?m)^\[`)
	nextMatch := nextSectionRe.FindStringIndex(content[start:])
	if nextMatch != nil {
		end := start + nextMatch[0]
		sectionContent := content[start:end]
		kvRe := regexp.MustCompile(`(?m)^(\w+)\s*=\s*(.+)$`)
		kvMatches := kvRe.FindAllStringSubmatch(sectionContent, -1)
		for _, kv := range kvMatches {
			key := kv[1]
			value := strings.Trim(strings.TrimSpace(kv[2]), "\"'")
			config[key] = value
		}
	} else {
		sectionContent := content[start:]
		kvRe := regexp.MustCompile(`(?m)^(\w+)\s*=\s*(.+)$`)
		kvMatches := kvRe.FindAllStringSubmatch(sectionContent, -1)
		for _, kv := range kvMatches {
			key := kv[1]
			value := strings.Trim(strings.TrimSpace(kv[2]), "\"'")
			config[key] = value
		}
	}

	return config
}

// parseChannelsConfig parses [channels_config.XXX] sections
func parseChannelsConfig(content string) map[string]ChannelConfig {
	channels := make(map[string]ChannelConfig)

	sectionRe := regexp.MustCompile(`(?m)^\[channels_config\.(\w+)\]`)
	matches := sectionRe.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		channelName := content[match[2]:match[3]]

		start := match[1]
		end := len(content)
		if i < len(matches)-1 {
			end = matches[i+1][0]
		}

		sectionContent := content[start:end]

		cfg := make(ChannelConfig)
		kvRe := regexp.MustCompile(`(?m)^(\w+)\s*=\s*(.+)$`)
		kvMatches := kvRe.FindAllStringSubmatch(sectionContent, -1)

		for _, kv := range kvMatches {
			key := kv[1]
			value := strings.Trim(strings.TrimSpace(kv[2]), "\"'")
			cfg[key] = value
		}

		if len(cfg) > 0 {
			channels[channelName] = cfg
		}
	}

	return channels
}

// GetDingTalkConfig returns DingTalk configuration
func (c *Config) GetDingTalkConfig() *DingTalkConfig {
	cfg, ok := c.Channels["dingtalk"]
	if !ok {
		return nil
	}

	dt := &DingTalkConfig{}
	dt.ClientID = cfg["client_id"]
	dt.ClientSecret = cfg["client_secret"]

	// Parse allowed_users
	if users, ok := cfg["allowed_users"]; ok {
		// Remove brackets and parse comma-separated values
		users = strings.Trim(users, "[]")
		for _, u := range strings.Split(users, ",") {
			u = strings.TrimSpace(strings.Trim(u, "\"'"))
			if u != "" {
				dt.AllowedUsers = append(dt.AllowedUsers, u)
			}
		}
	}

	return dt
}

// HasChannels returns true if any channel is configured
func (c *Config) HasChannels() bool {
	return len(c.Channels) > 0
}

// parseTomlString parses a string value from TOML content
func parseTomlString(content, key, defaultValue string) string {
	// Match: key = "value" or key = 'value' or key = value
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `\s*=\s*["']?([^"'\n\r]+)["']?\s*$`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return defaultValue
}

// parseTomlNestedString parses a nested key like "memory.backend" from TOML content
func parseTomlNestedString(content, key, defaultValue string) string {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return parseTomlString(content, key, defaultValue)
	}

	section := parts[0]
	field := parts[1]

	// Find the section first
	sectionRe := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\]`)
	sectionMatch := sectionRe.FindStringIndex(content)
	if sectionMatch == nil {
		return defaultValue
	}

	start := sectionMatch[1]

	// Find the next section
	nextSectionRe := regexp.MustCompile(`(?m)^\[`)
	nextMatch := nextSectionRe.FindStringIndex(content[start:])
	var sectionContent string
	if nextMatch != nil {
		end := start + nextMatch[0]
		sectionContent = content[start:end]
	} else {
		sectionContent = content[start:]
	}

	// Now find the field in the section
	fieldRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(field) + `\s*=\s*["']?([^"'\n\r]+)["']?\s*$`)
	fieldMatches := fieldRe.FindStringSubmatch(sectionContent)
	if len(fieldMatches) > 1 {
		return strings.TrimSpace(fieldMatches[1])
	}

	return defaultValue
}

// parseTomlInt parses an int value from TOML content
func parseTomlInt(content, key string, defaultValue int) int {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `\s*=\s*(\d+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		val, err := strconv.Atoi(matches[1])
		if err == nil {
			return val
		}
	}
	return defaultValue
}

// parseTomlFloat parses a float value from TOML content
func parseTomlFloat(content, key string, defaultValue float64) float64 {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `\s*=\s*([\d.]+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		val, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return val
		}
	}
	return defaultValue
}

func (c *Config) GetProvider() *ProviderConfig {
	return &c.Provider
}

// GetSkillsDir returns the skills directory path
func (c *Config) GetSkillsDir() string {
	if c.SkillsDir != "" {
		return c.SkillsDir
	}
	// Fallback to default
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		return filepath.Join(homeDir, ".goclaw", "workspace", "skills")
	}
	return "."
}

// GetAuth returns the authentication configuration
func (c *Config) GetAuth() *AuthConfig {
	return &c.Auth
}

// parseTomlNestedInt parses a nested integer key like "gateway.port" from TOML content
func parseTomlNestedInt(content, key string, defaultValue int) int {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return parseTomlInt(content, key, defaultValue)
	}

	section := parts[0]
	field := parts[1]

	sectionRe := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\]`)
	sectionMatch := sectionRe.FindStringIndex(content)
	if sectionMatch == nil {
		return defaultValue
	}

	start := sectionMatch[1]

	nextSectionRe := regexp.MustCompile(`(?m)^\[`)
	nextMatch := nextSectionRe.FindStringIndex(content[start:])
	var sectionContent string
	if nextMatch != nil {
		end := start + nextMatch[0]
		sectionContent = content[start:end]
	} else {
		sectionContent = content[start:]
	}

	fieldRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(field) + `\s*=\s*(\d+)`)
	fieldMatches := fieldRe.FindStringSubmatch(sectionContent)
	if len(fieldMatches) > 1 {
		val, err := strconv.Atoi(fieldMatches[1])
		if err == nil {
			return val
		}
	}

	return defaultValue
}

// parseTomlNestedStringArray parses a nested string array key like "gateway.paired_tokens" from TOML content
func parseTomlNestedStringArray(content, key string, defaultValue []string) []string {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return defaultValue
	}

	section := parts[0]
	field := parts[1]

	sectionRe := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\]`)
	sectionMatch := sectionRe.FindStringIndex(content)
	if sectionMatch == nil {
		return defaultValue
	}

	start := sectionMatch[1]

	nextSectionRe := regexp.MustCompile(`(?m)^\[`)
	nextMatch := nextSectionRe.FindStringIndex(content[start:])
	var sectionContent string
	if nextMatch != nil {
		end := start + nextMatch[0]
		sectionContent = content[start:end]
	} else {
		sectionContent = content[start:]
	}

	fieldRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(field) + `\s*=\s*\[(.+)\]`)
	fieldMatches := fieldRe.FindStringSubmatch(sectionContent)
	if len(fieldMatches) > 1 {
		arrayContent := strings.TrimSpace(fieldMatches[1])
		if arrayContent == "" {
			return []string{}
		}

		var result []string
		items := strings.Split(arrayContent, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			item = strings.Trim(item, "\"'")
			if item != "" {
				result = append(result, item)
			}
		}
		return result
	}

	return defaultValue
}
