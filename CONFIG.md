# GoClaw 配置文件说明

## 📁 配置文件位置

配置文件位于：`~/.goclaw/config.toml`

首次运行时会自动创建默认配置文件，也可以运行 `go run main.go onboard` 启动配置向导进行交互式配置。

## 🔧 配置文件结构

### 基础配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `default_provider` | string | "openai" | 默认 AI 提供商名称 |
| `default_model` | string | "gpt-4" | 默认使用的模型名称 |
| `api_key` | string | - | API 密钥（建议使用环境变量） |
| `base_url` | string | - | 自定义 API 基础 URL |
| `default_temperature` | float64 | 0.7 | 默认温度参数（0.0-2.0） |
| `skills_dir` | string | ~/.goclaw/workspace/skills | 技能目录路径 |
| `static_dir` | string | ~/.goclaw/web/dist | Web 界面静态文件目录 |

**示例：**

**使用 Gitee AI（推荐，免费模型）：**
```toml
default_provider = "custom:https://ai.gitee.com/v1"
default_model = "GLM-4.7-Flash"
api_key = "your-gitee-api-key"
```

**使用阿里云百炼（Coding Plan Lite 支持）：**
```toml
default_provider = "bailian"
default_model = "qwen-plus"
api_key = "your-bailian-api-key"
```

**使用 OpenAI：**
```toml
default_provider = "openai"
default_model = "gpt-4"
api_key = "your-openai-api-key"
```

**使用 Ollama 本地模型：**
```toml
default_provider = "ollama"
default_model = "llama3.1:8b"
ollama_host = "http://localhost:11434"
```

### [agent] - Agent 配置

Agent 行为和执行参数配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `max_tool_iterations` | int | 15 | 最大工具调用迭代次数，用于防止死循环 |
| `max_history_messages` | int | 20 | 最大历史消息数量，超过后会进行压缩 |
| `parallel_tools` | bool | false | 是否并行执行工具 |
| `tool_dispatcher` | string | "auto" | 工具调度器类型：auto, simple, advanced |
| `compact_context` | bool | true | 是否压缩上下文以节省 token |

**示例：**
```toml
[agent]
max_tool_iterations = 15
max_history_messages = 20
parallel_tools = false
tool_dispatcher = "auto"
compact_context = true
```

### [memory] - 记忆体配置

记忆体存储和检索配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `backend` | string | "sqlite" | 记忆体后端：none, sqlite, qdrant |
| `auto_save` | bool | true | 是否自动保存对话到记忆体 |
| `hygiene_enabled` | bool | true | 是否启用记忆体清理 |
| `archive_after_days` | int | 7 | 归档天数，超过此天数的对话会被归档 |
| `purge_after_days` | int | 30 | 清理天数，超过此天数的对话会被删除 |
| `conversation_retention_days` | int | 30 | 对话保留天数 |
| `embedding_provider` | string | "none" | 嵌入向量提供商 |
| `embedding_model` | string | "text-embedding-3-small" | 嵌入模型 |
| `embedding_dimensions` | int | 1536 | 嵌入向量维度 |
| `vector_weight` | float64 | 0.7 | 向量搜索权重（0.0-1.0） |
| `keyword_weight` | float64 | 0.3 | 关键词搜索权重（0.0-1.0） |
| `min_relevance_score` | float64 | 0.4 | 最小相关性分数（0.0-1.0） |
| `embedding_cache_size` | int | 10000 | 嵌入缓存大小 |
| `chunk_max_tokens` | int | 512 | 文本分块最大 token 数 |
| `response_cache_enabled` | bool | false | 是否启用响应缓存 |
| `response_cache_ttl_minutes` | int | 60 | 响应缓存 TTL（分钟） |
| `response_cache_max_entries` | int | 5000 | 响应缓存最大条目数 |
| `snapshot_enabled` | bool | false | 是否启用快照 |
| `snapshot_on_hygiene` | bool | false | 清理时是否创建快照 |
| `auto_hydrate` | bool | true | 是否自动清理过期数据 |

**示例：**
```toml
[memory]
backend = "sqlite"
auto_save = true
hygiene_enabled = true
archive_after_days = 7
purge_after_days = 30
vector_weight = 0.7
keyword_weight = 0.3
min_relevance_score = 0.4
```

### [gateway] - 网关配置

Web 网关服务器配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `port` | int | 4096 | 网关服务器端口 |
| `host` | string | "0.0.0.0" | 网关服务器监听地址 |
| `locale` | string | "zh-CN" | 界面语言 |
| `require_pairing` | bool | false | 是否需要配对码 |
| `allow_public_bind` | bool | false | 是否允许公网绑定 |
| `paired_tokens` | array | [] | 已配对的 token 列表 |
| `pair_rate_limit_per_minute` | int | 10 | 配对速率限制（每分钟） |
| `webhook_rate_limit_per_minute` | int | 60 | Webhook 速率限制（每分钟） |
| `trust_forwarded_headers` | bool | false | 是否信任转发的请求头 |
| `rate_limit_max_keys` | int | 10000 | 速率限制最大键数 |
| `idempotency_ttl_secs` | int | 300 | 幂等性 TTL（秒） |
| `idempotency_max_keys` | int | 10000 | 幂等性最大键数 |

**示例：**
```toml
[gateway]
port = 4096
host = "0.0.0.0"
locale = "zh-CN"
require_pairing = false
```

### [wechat] - 微信登录配置

微信扫码登录功能配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用微信登录 |
| `app_id` | string | - | 微信开放平台 App ID |
| `app_secret` | string | - | 微信开放平台 App Secret |
| `redirect_uri` | string | - | 微信登录回调地址 |

**示例：**
```toml
[wechat]
enabled = true
app_id = "your-wechat-app-id"
app_secret = "your-wechat-app-secret"
redirect_uri = "https://your-domain.com/auth/wechat/callback"
```

### [auth] - 认证配置

用户认证和权限管理配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enable_login` | bool | false | 是否启用登录功能 |
| `enable_audit` | bool | false | 是否启用管理员审核（新用户需审核后才能使用） |

**示例：**
```toml
[auth]
enable_login = true
enable_audit = true
```

### [skills] - 技能配置

技能系统配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `open_skills_enabled` | bool | false | 是否启用开放技能 |
| `prompt_injection_mode` | string | "full" | 提示注入模式：none, partial, full |

**示例：**
```toml
[skills]
open_skills_enabled = false
prompt_injection_mode = "full"
```

### [channels_config] - 通知渠道配置

各种通知渠道的配置。

#### [channels_config.cli] - CLI 渠道

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `cli` | bool | true | 是否启用 CLI 渠道 |
| `message_timeout_secs` | int | 60 | 消息超时时间（秒） |

#### [channels_config.email] - 邮件渠道

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用邮件渠道 |
| `imap_host` | string | - | IMAP 服务器地址 |
| `imap_port` | int | 993 | IMAP 端口 |
| `imap_folder` | string | "INBOX" | IMAP 文件夹 |
| `smtp_host` | string | - | SMTP 服务器地址 |
| `smtp_port` | int | 465 | SMTP 端口 |
| `smtp_tls` | bool | true | 是否使用 TLS |
| `username` | string | - | 邮箱用户名 |
| `password` | string | - | 邮箱密码 |
| `from_address` | string | - | 发件人地址 |
| `idle_timeout_secs` | int | 30 | 空闲超时（秒） |
| `disable_idle` | bool | true | 是否禁用空闲检查 |
| `allowed_senders` | array | [] | 允许的发送者列表 |

**示例：**
```toml
[channels_config.email]
enabled = true
imap_host = "imap.qq.com"
imap_port = 993
imap_folder = "INBOX"
smtp_host = "smtp.qq.com"
smtp_port = 465
smtp_tls = true
username = "your-email@qq.com"
password = "your-password"
from_address = "your-email@qq.com"
```

#### [channels_config.dingtalk] - 钉钉渠道

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `client_id` | string | - | 钉钉 Client ID |
| `client_secret` | string | - | 钉钉 Client Secret |
| `allowed_users` | array | [] | 允许的用户列表（["*"] 表示所有用户） |

**示例：**
```toml
[channels_config.dingtalk]
client_id = "your-client-id"
client_secret = "your-client-secret"
allowed_users = ["*"]
```

#### [channels_config.wecom] - 企业微信渠道

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enable` | bool | false | 是否启用企业微信渠道 |
| `bot_id` | string | - | 企业微信机器人 ID |
| `bot_secret` | string | - | 企业微信机器人 Secret |
| `allowed_users` | array | [] | 允许的用户列表（["*"] 表示所有用户） |
| `default_to` | string | - | 默认消息接收人（当没有指定接收人时使用） |

**示例：**
```toml
[channels_config.wecom]
enable = true
bot_id = "aibjVtI1HRyG-LDINkpFYDvXuIccnTzp7Ig"
bot_secret = "WFhq0icnlrRfav1XaGEOIKOjvqtyU3MLs37CLwjCp5q"
allowed_users = ["*"]
default_to = "your-default-recipient"
```

**功能特性：**
- ✅ WebSocket 长连接，实时接收消息
- ✅ 支持单聊和群聊
- ✅ 流式回复，打字效果
- ✅ 自动去除消息前的 @机器人标记
- ✅ 支持文本、图片、语音、文件等多种消息类型

**获取机器人凭证：**
1. 登录企业微信管理后台：https://work.weixin.qq.com/
2. 进入「应用管理」→「应用」→「智能机器人」
3. 创建或选择机器人应用
4. 在应用详情页获取 `bot_id` 和 `bot_secret`
5. 将机器人添加到群聊中（群聊需要 @机器人才能触发）

**注意事项：**
- 企业微信机器人需要通过 WebSocket 长连接接收消息
- 群聊中需要 @机器人才能触发消息推送
- 机器人头像需要在企业微信管理后台设置，代码无法控制
- 消息回复使用流式接口，模拟打字效果（每 10 个字符延迟 100 毫秒）

### [scheduler] - 调度器配置

任务调度器配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | true | 是否启用调度器 |
| `max_tasks` | int | 64 | 最大任务数 |
| `max_concurrent` | int | 4 | 最大并发任务数 |

**示例：**
```toml
[scheduler]
enabled = true
max_tasks = 64
max_concurrent = 4
```

### [cron] - 定时任务配置

定时任务系统配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | true | 是否启用定时任务 |
| `max_run_history` | int | 50 | 最大运行历史记录数 |

**示例：**
```toml
[cron]
enabled = true
max_run_history = 50
```

### [reliability] - 可靠性配置

系统可靠性和容错配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `provider_retries` | int | 1 | 提供商重试次数 |
| `provider_backoff_ms` | int | 200 | 提供商退避时间（毫秒） |
| `fallback_providers` | array | [] | 备用提供商列表 |
| `api_keys` | array | [] | API 密钥列表（用于轮询） |
| `channel_initial_backoff_secs` | int | 1 | 渠道初始退避时间（秒） |
| `channel_max_backoff_secs` | int | 10 | 渠道最大退避时间（秒） |
| `scheduler_poll_secs` | int | 2 | 调度器轮询间隔（秒） |
| `scheduler_retries` | int | 1 | 调度器重试次数 |

**示例：**
```toml
[reliability]
provider_retries = 1
provider_backoff_ms = 200
fallback_providers = []
channel_initial_backoff_secs = 1
channel_max_backoff_secs = 10
```

### [cost] - 成本控制配置

API 调用成本控制。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用成本控制 |
| `daily_limit_usd` | float64 | 10.0 | 每日成本限制（美元） |
| `monthly_limit_usd` | float64 | 100.0 | 每月成本限制（美元） |
| `warn_at_percent` | int | 80 | 警告阈值（百分比） |
| `allow_override` | bool | false | 是否允许覆盖限制 |

**示例：**
```toml
[cost]
enabled = false
daily_limit_usd = 10.0
monthly_limit_usd = 100.0
warn_at_percent = 80
```

### [web_search] - 网络搜索配置

网络搜索功能配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用网络搜索 |
| `provider` | string | "duckduckgo" | 搜索提供商 |
| `max_results` | int | 5 | 最大结果数 |
| `timeout_secs` | int | 15 | 超时时间（秒） |

**示例：**
```toml
[web_search]
enabled = false
provider = "duckduckgo"
max_results = 5
timeout_secs = 15
```

### [web_fetch] - 网页获取配置

网页内容获取配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用网页获取 |
| `allowed_domains` | array | ["*"] | 允许的域名列表 |
| `blocked_domains` | array | [] | 阻止的域名列表 |
| `max_response_size` | int | 500000 | 最大响应大小（字节） |
| `timeout_secs` | int | 30 | 超时时间（秒） |

**示例：**
```toml
[web_fetch]
enabled = false
allowed_domains = ["*"]
blocked_domains = []
max_response_size = 500000
timeout_secs = 30
```

### [browser] - 浏览器配置

浏览器功能配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用浏览器 |
| `allowed_domains` | array | [] | 允许的域名列表 |
| `backend` | string | "agent_browser" | 浏览器后端 |
| `native_headless` | bool | true | 是否使用无头模式 |
| `native_webdriver_url` | string | - | 原生 WebDriver URL |

**示例：**
```toml
[browser]
enabled = false
allowed_domains = []
backend = "agent_browser"
native_headless = true
```

### [http_request] - HTTP 请求配置

HTTP 请求功能配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | true | 是否启用 HTTP 请求 |
| `allowed_domains` | array | [...] | 允许的域名列表 |
| `max_response_size` | int | 1000000 | 最大响应大小（字节） |
| `timeout_secs` | int | 30 | 超时时间（秒） |

**示例：**
```toml
[http_request]
enabled = true
allowed_domains = ["oapi.dingtalk.com", "api.tianqiapi.com"]
max_response_size = 1000000
timeout_secs = 30
```

### [heartbeat] - 心跳引擎配置

心跳事件引擎配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用心跳引擎 |
| `interval_minutes` | int | 30 | 心跳间隔（分钟） |

**示例：**
```toml
[heartbeat]
enabled = false
interval_minutes = 30
```

### [hooks] - 钩子配置

事件钩子系统配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | true | 是否启用钩子系统 |

**内置钩子配置：**
```toml
[hooks]
enabled = true

[hooks.builtin]
command_logger = false
```

**可用事件类型：**
- `pre_tool_exec` - 工具执行前
- `post_tool_exec` - 工具执行后
- `pre_agent_loop` - Agent 循环前
- `post_agent_loop` - Agent 循环后
- `on_error` - 错误发生时
- `on_message` - 消息处理时

### [autonomy] - 自主性配置

Agent 自主性和安全配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `level` | string | "supervised" | 自主性级别：full, supervised, approve, read_only, disabled |
| `workspace_only` | bool | true | 是否限制在工作空间内 |
| `allowed_commands` | array | [...] | 允许执行的命令列表 |
| `forbidden_paths` | array | [...] | 禁止访问的路径列表 |
| `max_actions_per_hour` | int | 20 | 每小时最大操作数 |
| `max_cost_per_day_cents` | int | 500 | 每日最大成本（美分） |
| `require_approval_for_medium_risk` | bool | false | 中等风险操作是否需要审批 |
| `block_high_risk_commands` | bool | true | 是否阻止高风险命令 |
| `auto_approve` | array | [...] | 自动批准的工具列表 |
| `always_ask` | array | [...] | 总是询问的工具列表 |

**示例：**
```toml
[autonomy]
level = "supervised"
workspace_only = true
allowed_commands = ["git", "npm", "cargo", "ls", "cat", "grep", "find"]
forbidden_paths = ["/etc", "/root", "/home", "~/.ssh", "~/.aws"]
max_actions_per_hour = 20
max_cost_per_day_cents = 500
```

### [security] - 安全配置

安全策略和沙箱配置。

#### [security.sandbox]

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `backend` | string | "auto" | 沙箱后端：auto, firejail, none |
| `firejail_args` | array | [] | Firejail 额外参数 |

#### [security.resources]

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `max_memory_mb` | int | 512 | 最大内存（MB） |
| `max_cpu_time_seconds` | int | 60 | 最大 CPU 时间（秒） |
| `max_subprocesses` | int | 10 | 最大子进程数 |
| `memory_monitoring` | bool | true | 是否启用内存监控 |

#### [security.audit]

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | true | 是否启用审计日志 |
| `log_path` | string | "audit.log" | 审计日志路径 |
| `max_size_mb` | int | 100 | 日志最大大小（MB） |

**示例：**
```toml
[security.sandbox]
backend = "auto"

[security.resources]
max_memory_mb = 512
max_cpu_time_seconds = 60

[security.audit]
enabled = true
log_path = "audit.log"
```

### [runtime] - 运行时配置

运行时环境配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `kind` | string | "native" | 运行时类型：native, docker, wasm |

**Docker 运行时配置：**
```toml
[runtime]
kind = "docker"

[runtime.docker]
image = "alpine:3.20"
network = "none"
memory_limit_mb = 512
cpu_limit = 1.0
read_only_rootfs = true
mount_workspace = true
```

### [observability] - 可观测性配置

可观测性配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `backend` | string | "none" | 可观测性后端 |
| `runtime_trace_mode` | string | "none" | 运行时追踪模式 |
| `runtime_trace_path` | string | "state/runtime-trace.jsonl" | 追踪文件路径 |
| `runtime_trace_max_entries` | int | 200 | 最大追踪条目数 |

### [multimodal] - 多模态配置

多模态功能配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `max_images` | int | 4 | 最大图片数 |
| `max_image_size_mb` | int | 5 | 最大图片大小（MB） |
| `allow_remote_fetch` | bool | false | 是否允许远程获取图片 |

### [transcription] - 语音转录配置

语音转文字配置。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用语音转录 |
| `api_url` | string | "https://api.groq.com/openai/v1/audio/transcriptions" | 转录 API 地址 |
| `model` | string | "whisper-large-v3-turbo" | 转录模型 |
| `max_duration_secs` | int | 120 | 最大音频时长（秒） |

## 🎯 常见配置场景

### 场景 1：多步骤任务（股票分析 + 邮件发送）

对于需要多次工具调用的复杂任务，建议增加最大迭代次数：

```toml
[agent]
max_tool_iterations = 20
```

### 场景 2：优化记忆体检索

提高记忆体检索准确率：

```toml
[memory]
vector_weight = 0.7
keyword_weight = 0.3
min_relevance_score = 0.3
```

### 场景 3：启用微信登录

配置微信扫码登录：

```toml
[auth]
enable_login = true
enable_audit = true

[wechat]
enabled = true
app_id = "your-wechat-app-id"
app_secret = "your-wechat-app-secret"
redirect_uri = "https://your-domain.com/auth/wechat/callback"
```

### 场景 4：启用邮件通知

配置邮件渠道：

```toml
[channels_config.email]
enabled = true
smtp_host = "smtp.qq.com"
smtp_port = 465
smtp_tls = true
username = "your-email@qq.com"
password = "your-password"
from_address = "your-email@qq.com"
```

### 场景 5：成本控制

启用 API 成本控制：

```toml
[cost]
enabled = true
daily_limit_usd = 10.0
monthly_limit_usd = 100.0
warn_at_percent = 80
```

### 场景 6：钉钉机器人集成

配置钉钉机器人：

```toml
[channels_config.dingtalk]
client_id = "your-dingtalk-client-id"
client_secret = "your-dingtalk-client-secret"
allowed_users = ["*"]
```

## 🔐 环境变量

除了配置文件，也可以使用环境变量：

| 环境变量 | 说明 |
|----------|------|
| `BAILIAN_API_KEY` | 阿里云百炼 API 密钥 |
| `OPENAI_API_KEY` | OpenAI API 密钥 |
| `GITEE_AI_API_KEY` | GiteeAI API 密钥 |
| `ANTHROPIC_API_KEY` | Anthropic API 密钥 |
| `GEMINI_API_KEY` | Google Gemini API 密钥 |

## 📝 配置文件示例

### 最小配置（Gitee AI 免费）

```toml
default_provider = "custom:https://ai.gitee.com/v1"
default_model = "GLM-4.7-Flash"
api_key = "your-gitee-api-key"

[gateway]
port = 4096
host = "127.0.0.1"
```

### 完整配置示例

```toml
# GoClaw Configuration

# AI 提供商配置
default_provider = "custom:https://ai.gitee.com/v1"
default_model = "GLM-4.7-Flash"
api_key = "your-api-key"
default_temperature = 0.7

[agent]
max_tool_iterations = 15
max_history_messages = 20
parallel_tools = false
tool_dispatcher = "auto"
compact_context = true

[memory]
backend = "sqlite"
auto_save = true
hygiene_enabled = true
archive_after_days = 7
purge_after_days = 30
vector_weight = 0.7
keyword_weight = 0.3
min_relevance_score = 0.4

[gateway]
port = 4096
host = "0.0.0.0"
locale = "zh-CN"
require_pairing = false

# 微信登录配置
[wechat]
enabled = true
app_id = "your-wechat-app-id"
app_secret = "your-wechat-app-secret"
redirect_uri = "https://your-domain.com/auth/wechat/callback"

[auth]
enable_login = true
enable_audit = true

[skills]
open_skills_enabled = false
prompt_injection_mode = "full"

[scheduler]
enabled = true
max_tasks = 64
max_concurrent = 4

[cron]
enabled = true
max_run_history = 50

[reliability]
provider_retries = 1
provider_backoff_ms = 200
channel_initial_backoff_secs = 1
channel_max_backoff_secs = 10

[cost]
enabled = false
daily_limit_usd = 10.0
monthly_limit_usd = 100.0
warn_at_percent = 80

[web_search]
enabled = false
provider = "duckduckgo"
max_results = 5
timeout_secs = 15

[web_fetch]
enabled = false
allowed_domains = ["*"]
blocked_domains = []
max_response_size = 500000
timeout_secs = 30

[browser]
enabled = false
allowed_domains = []
backend = "agent_browser"
native_headless = true

[http_request]
enabled = true
allowed_domains = ["oapi.dingtalk.com", "api.tianqiapi.com"]
max_response_size = 1000000
timeout_secs = 30

[heartbeat]
enabled = false
interval_minutes = 30

[hooks]
enabled = true

[hooks.builtin]
command_logger = false

[autonomy]
level = "supervised"
workspace_only = true
allowed_commands = ["git", "npm", "cargo", "ls", "cat", "grep"]
forbidden_paths = ["/etc", "/root", "~/.ssh", "~/.aws"]

[security.sandbox]
backend = "auto"

[security.resources]
max_memory_mb = 512
max_cpu_time_seconds = 60

[security.audit]
enabled = true
log_path = "audit.log"

[runtime]
kind = "native"

[observability]
backend = "none"

[multimodal]
max_images = 4
max_image_size_mb = 5

[transcription]
enabled = false
```

## 🔄 配置重载

修改配置文件后，需要重启服务才能生效：

```bash
# 停止服务
lsof -ti:4096 | xargs kill -9

# 启动服务
go run main.go gateway
```

## 🚀 配置向导

首次使用可以运行配置向导进行交互式配置：

```bash
go run main.go onboard
```

配置向导会引导你：
1. 选择 AI 提供商（OpenAI、Anthropic、Gemini、GLM、Ollama、GiteeAI、百炼）
2. 输入 API 密钥
3. 选择模型
4. 选择存储后端（无、SQLite、Qdrant）
5. 选择要启用的通知渠道
6. 设置工作空间目录

## 📚 相关文档

- [README.md](readme.md) - 项目说明
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - 系统架构
- [docs/API.md](docs/API.md) - API 文档
- [docs/NODES_IMPLEMENTATION.md](docs/NODES_IMPLEMENTATION.md) - 节点实现

## 💡 配置建议

1. **安全性**：不要在配置文件中存储敏感信息，使用环境变量
2. **性能**：根据任务复杂度调整 `max_tool_iterations`
3. **成本**：启用成本控制以避免意外费用
4. **可靠性**：配置备用提供商和重试策略
5. **记忆体**：调整向量权重和相关性分数以优化检索效果
6. **登录认证**：生产环境建议启用登录功能保护 Web 界面

## 🆘 问题排查

### 配置不生效

1. 检查配置文件路径：`~/.goclaw/config.toml`
2. 确认配置文件格式正确（TOML 语法）
3. 重启服务以加载新配置
4. 查看服务启动日志确认配置已加载

### 工具调用次数不足

1. 增加 `max_tool_iterations` 值
2. 检查是否有死循环（重复的工具调用）
3. 优化 Agent 提示词以减少不必要的工具调用

### 记忆体检索不准确

1. 调整 `vector_weight` 和 `keyword_weight` 比例
2. 降低 `min_relevance_score` 以包含更多结果
3. 检查记忆体内容是否包含相关关键词

### 微信登录失败

1. 检查微信开放平台配置是否正确
2. 确认回调地址与配置中的 `redirect_uri` 一致
3. 确保 App ID 和 App Secret 正确

### Web 界面无法访问

1. 检查 `gateway.port` 和 `gateway.host` 配置
2. 确认防火墙没有阻止端口
3. 如果启用登录，确保认证配置正确