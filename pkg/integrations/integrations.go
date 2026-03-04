// Package integrations provides integration catalog for GoClaw.
package integrations

// IntegrationCategory represents the category of an integration.
type IntegrationCategory string

const (
	CategoryChat           IntegrationCategory = "Chat"
	CategoryAiModel        IntegrationCategory = "AiModel"
	CategoryProductivity   IntegrationCategory = "Productivity"
	CategoryMusicAudio     IntegrationCategory = "MusicAudio"
	CategorySmartHome      IntegrationCategory = "SmartHome"
	CategoryToolsAutomation IntegrationCategory = "ToolsAutomation"
	CategoryMediaCreative  IntegrationCategory = "MediaCreative"
	CategorySocial         IntegrationCategory = "Social"
	CategoryPlatform       IntegrationCategory = "Platform"
)

// IntegrationStatus represents the status of an integration.
type IntegrationStatus string

const (
	StatusActive     IntegrationStatus = "Active"
	StatusAvailable  IntegrationStatus = "Available"
	StatusComingSoon IntegrationStatus = "ComingSoon"
)

// IntegrationEntry represents a single integration.
type IntegrationEntry struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Category    IntegrationCategory `json:"category"`
	Status      IntegrationStatus  `json:"status"`
}

// allIntegrations returns the full catalog of integrations.
func allIntegrations() []IntegrationEntry {
	return []IntegrationEntry{
		// ── Chat Providers ──────────────────────────────────────
		{Name: "Telegram", Description: "机器人 API — 长轮询", Category: CategoryChat, Status: StatusAvailable},
		{Name: "Discord", Description: "服务器、频道和私信", Category: CategoryChat, Status: StatusAvailable},
		{Name: "Slack", Description: "通过 Web API 的工作区应用", Category: CategoryChat, Status: StatusAvailable},
		{Name: "Webhooks", Description: "触发器的 HTTP 端点", Category: CategoryChat, Status: StatusAvailable},
		{Name: "WhatsApp", Description: "通过 webhook 的 Meta Cloud API", Category: CategoryChat, Status: StatusAvailable},
		{Name: "Signal", Description: "通过 signal-cli 的隐私优先", Category: CategoryChat, Status: StatusAvailable},
		{Name: "iMessage", Description: "macOS AppleScript 桥接", Category: CategoryChat, Status: StatusAvailable},
		{Name: "Microsoft Teams", Description: "企业聊天支持", Category: CategoryChat, Status: StatusComingSoon},
		{Name: "Matrix", Description: "Matrix 协议（Element）", Category: CategoryChat, Status: StatusAvailable},
		{Name: "Nostr", Description: "去中心化私信（NIP-04）", Category: CategoryChat, Status: StatusComingSoon},
		{Name: "WebChat", Description: "基于浏览器的聊天界面", Category: CategoryChat, Status: StatusComingSoon},
		{Name: "Nextcloud Talk", Description: "自托管的 Nextcloud 聊天", Category: CategoryChat, Status: StatusComingSoon},
		{Name: "Zalo", Description: "Zalo 机器人 API", Category: CategoryChat, Status: StatusComingSoon},
		{Name: "DingTalk", Description: "钉钉流模式", Category: CategoryChat, Status: StatusAvailable},
		{Name: "QQ Official", Description: "腾讯 QQ 机器人 SDK", Category: CategoryChat, Status: StatusAvailable},

		// ── AI Models ───────────────────────────────────────────
		{Name: "OpenRouter", Description: "Claude Sonnet 4.6、GPT-5.2、Gemini 3.1 Pro", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Anthropic", Description: "Claude Sonnet 4.6、Claude Opus 4.6", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "OpenAI", Description: "GPT-5.2、GPT-5.2-Codex", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Google", Description: "Gemini 3.1 Pro、Gemini 3 Flash", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "DeepSeek", Description: "DeepSeek-Reasoner、DeepSeek-Chat", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "xAI", Description: "Grok 4、Grok 3", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Mistral", Description: "Mistral Large 最新版、Codestral", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Ollama", Description: "本地模型（Llama 等）", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Perplexity", Description: "Sonar Pro、Sonar Reasoning Pro", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Hugging Face", Description: "开源模型", Category: CategoryAiModel, Status: StatusComingSoon},
		{Name: "LM Studio", Description: "本地模型服务器", Category: CategoryAiModel, Status: StatusComingSoon},
		{Name: "Venice", Description: "Venice Llama 3.3 70B 和前沿混合模型", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Vercel AI", Description: "GPT-5.2 和多提供商路由的网关", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Cloudflare AI", Description: "Workers AI + Llama 3.3 / 网关路由", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Moonshot", Description: "Kimi 2.5 和 Kimi Coding", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Synthetic", Description: "Synthetic-1 和 synthetic 系列模型", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "OpenCode Zen", Description: "OpenCode Zen 和编码专用模型", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Z.AI", Description: "GLM 4.7 和 Z.AI 托管变体", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "GLM", Description: "GLM 4.7 和 GLM 4.5 系列", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "MiniMax", Description: "MiniMax M1 和最新的多模态变体", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Qwen", Description: "Qwen Max 和 Qwen 推理系列", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Amazon Bedrock", Description: "Claude Sonnet 4.5 和 Bedrock 模型目录", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Qianfan", Description: "ERNIE 4.x 和千帆模型目录", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Groq", Description: "Llama 3.3 70B 多功能和低延迟模型", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Together AI", Description: "Llama 3.3 70B Turbo 和开源模型托管", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Fireworks AI", Description: "DeepSeek / Llama 高吞吐量推理", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Cohere", Description: "Command R+ (2024年8月) 和嵌入模型", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Bailian", Description: "阿里云百炼、通义千问系列模型、Coding Plan Lite", Category: CategoryAiModel, Status: StatusAvailable},
		{Name: "Gitee AI", Description: "Gitee AI 平台、GLM 系列模型", Category: CategoryAiModel, Status: StatusAvailable},

		// ── Productivity ────────────────────────────────────────
		{Name: "GitHub", Description: "代码、问题、PR", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Notion", Description: "工作区和数据库", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Apple Notes", Description: "原生 macOS/iOS 笔记", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Apple Reminders", Description: "任务管理", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Obsidian", Description: "知识图谱笔记", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Things 3", Description: "GTD 任务管理器", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Bear Notes", Description: "Markdown 笔记", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Trello", Description: "看板", Category: CategoryProductivity, Status: StatusComingSoon},
		{Name: "Linear", Description: "问题跟踪", Category: CategoryProductivity, Status: StatusComingSoon},

		// ── Music & Audio ───────────────────────────────────────
		{Name: "Spotify", Description: "音乐播放控制", Category: CategoryMusicAudio, Status: StatusComingSoon},
		{Name: "Sonos", Description: "多房间音频", Category: CategoryMusicAudio, Status: StatusComingSoon},
		{Name: "Shazam", Description: "歌曲识别", Category: CategoryMusicAudio, Status: StatusComingSoon},

		// ── Smart Home ──────────────────────────────────────────
		{Name: "Home Assistant", Description: "家庭自动化中心", Category: CategorySmartHome, Status: StatusComingSoon},
		{Name: "Philips Hue", Description: "智能照明", Category: CategorySmartHome, Status: StatusComingSoon},
		{Name: "8Sleep", Description: "智能床垫", Category: CategorySmartHome, Status: StatusComingSoon},

		// ── Tools & Automation ──────────────────────────────────
		{Name: "Browser", Description: "Chrome/Chromium 控制", Category: CategoryToolsAutomation, Status: StatusAvailable},
		{Name: "Shell", Description: "终端命令执行", Category: CategoryToolsAutomation, Status: StatusActive},
		{Name: "File System", Description: "读写文件", Category: CategoryToolsAutomation, Status: StatusActive},
		{Name: "Cron", Description: "计划任务", Category: CategoryToolsAutomation, Status: StatusAvailable},
		{Name: "Voice", Description: "语音唤醒 + 对话模式", Category: CategoryToolsAutomation, Status: StatusComingSoon},
		{Name: "Gmail", Description: "邮件触发器和发送", Category: CategoryToolsAutomation, Status: StatusComingSoon},
		{Name: "1Password", Description: "安全凭证", Category: CategoryToolsAutomation, Status: StatusComingSoon},
		{Name: "Weather", Description: "天气预报和状况", Category: CategoryToolsAutomation, Status: StatusComingSoon},
		{Name: "Canvas", Description: "可视化工作区 + A2UI", Category: CategoryToolsAutomation, Status: StatusComingSoon},

		// ── Media & Creative ────────────────────────────────────
		{Name: "Image Gen", Description: "AI 图像生成", Category: CategoryMediaCreative, Status: StatusComingSoon},
		{Name: "GIF Search", Description: "查找完美的 GIF", Category: CategoryMediaCreative, Status: StatusComingSoon},
		{Name: "Screen Capture", Description: "截图和屏幕控制", Category: CategoryMediaCreative, Status: StatusComingSoon},
		{Name: "Camera", Description: "照片/视频捕获", Category: CategoryMediaCreative, Status: StatusComingSoon},

		// ── Social ──────────────────────────────────────────────
		{Name: "Twitter/X", Description: "发推文、回复、搜索", Category: CategorySocial, Status: StatusComingSoon},
		{Name: "Email", Description: "IMAP/SMTP 邮件频道", Category: CategorySocial, Status: StatusAvailable},

		// ── Platform ────────────────────────────────────────────
		{Name: "macOS", Description: "原生支持 + AppleScript", Category: CategoryPlatform, Status: StatusActive},
		{Name: "Linux", Description: "原生支持", Category: CategoryPlatform, Status: StatusAvailable},
		{Name: "Windows", Description: "推荐 WSL2", Category: CategoryPlatform, Status: StatusAvailable},
		{Name: "iOS", Description: "通过 Telegram/Discord 聊天", Category: CategoryPlatform, Status: StatusAvailable},
		{Name: "Android", Description: "通过 Telegram/Discord 聊天", Category: CategoryPlatform, Status: StatusAvailable},
	}
}

// ListIntegrations returns all integrations, optionally filtered by category or status.
func ListIntegrations(category *IntegrationCategory, status *IntegrationStatus) []IntegrationEntry {
	result := make([]IntegrationEntry, 0)
	for _, entry := range allIntegrations() {
		if category != nil && entry.Category != *category {
			continue
		}
		if status != nil && entry.Status != *status {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// GetAllIntegrations returns all integrations without filtering.
func GetAllIntegrations() []IntegrationEntry {
	return allIntegrations()
}
