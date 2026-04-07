# GoClaw
![logo-image](https://github.com/user-attachments/assets/c81a3f9e-c565-4dfb-96f4-39e4c40c07a6)


[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8E?style=flat&logo=go)](https://golang.org/)
[![GitHub Stars](https://img.shields.io/github/stars/lj7788/GoClaw?style=social)](https://github.com/lj7788/GoClaw/stargazers)

GoClaw 是基于 ZeroClaw 的 Go 语言实现版本，是一个功能强大、轻量高效的 AI 助手框架，并对 Web 端进行了中文化处理。

## ✨ 特性

- 🚀 **高性能**：基于 Go 语言开发，内存占用低，响应速度快
- 🌐 **多模型支持**：支持 Gitee AI、阿里云百炼、OpenAI、Anthropic、Gemini 等多种模型
- 📧 **邮件发送**：内置邮件发送技能，支持 SMTP 协议
- 📊 **股票分析**：集成股票分析技能，支持 A股、港股、美股实时行情
- 💾 **多存储后端**：支持 SQLite、Qdrant 等多种记忆存储
- 🔌 **多渠道集成**：支持企业微信、钉钉、微信、Telegram、Slack、Discord、Signal、iMessage 等 15+ 种通知渠道
- 🎨 **现代化 Web 界面**：使用 Vue 3 重写，支持中文界面
- 🛠️ **丰富的工具集**：内置文件操作、Web 搜索、Git 操作等工具
- 💬 **流式回复**：企业微信/飞书/钉钉支持流式回复，实时展示 AI 生成过程
- 🤖 **Hands 多智能体系统**：支持多智能体协同工作
- 🔍 **SkillForge 技能工厂**：自动发现和集成 GitHub 技能
- 📚 **RAG 检索增强**：支持硬件手册知识库检索
- 🛡️ **Verifiable Intent**：企业级安全凭证系统
- 🎙️ **语音唤醒**：支持语音激活 AI 助手
- 🌐 **分布式架构**：支持多节点部署和隧道连接
- 🔧 **WASM 插件系统**：支持 WASM 插件扩展
- 🔗 **MCP 集成**：支持 Model Context Protocol
- 📈 **完整可观测性**：支持 OpenTelemetry、Prometheus、分布式追踪
- ✅ **企业工具集成**：支持 Jira、Notion、Microsoft 365、Google Workspace

## 📦 安装

### 前置要求

- Go 1.21 或更高版本
- Node.js 14 或更高版本（用于 Web 界面）
- Python 3.8 或更高版本（用于某些技能）

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/lj7788/GoClaw.git
cd GoClaw

# 下载依赖
go mod download

# 编译
go build -o bin/goclaw cmd/main.go

# 运行
./bin/goclaw daemon
```

### 配置技能

将 `skills` 目录中的技能复制到 `~/.goclaw/workspace/skills/` 目录：

```bash
# 邮件发送技能
cp -r skills/email-sender-skill ~/.goclaw/workspace/skills/

# 股票分析技能
cp -r skills/stock-analyzer-skill ~/.goclaw/workspace/skills/
```

#### 配置邮件发送技能

编辑 `~/.goclaw/workspace/skills/email-sender-skill/config.json`：

```json
{
  "smtp": {
    "host": "smtp.126.com",
    "port": 465,
    "secure": true,
    "auth": {
      "user": "your-email@126.com",
      "pass": "your-auth-code"
    }
  }
}
```

#### 配置 AI 模型提供商

编辑 `~/.goclaw/config.toml` 文件选择 AI 模型提供商：

**使用阿里云百炼（推荐，支持 Coding Plan Lite）：**

```toml
default_provider = "bailian"
default_model = "qwen-plus"
api_key = "your-bailian-api-key"
```

支持的百炼模型：
- `qwen-plus` - 通义千问 Plus 模型
- `qwen-coder-plus` - 通义千问 Coder Plus 模型（适合编程）
- `qwen-coder-turbo` - 通义千问 Coder Turbo 模型（快速编程）
- `qwen-max` - 通义千问 Max 模型（最强性能）
- `qwen-turbo` - 通义千问 Turbo 模型（快速响应）
- `qwen-flash` - 通义千问 Flash 模型（极速响应）
- `glm-4-flash` - GLM-4 Flash 模型
- `glm-4-plus` - GLM-4 Plus 模型
- `glm-4` - GLM-4 模型
- `glm-4v-flash` - GLM-4V Flash 视觉模型
- `glm-4v-plus` - GLM-4V Plus 视觉模型

获取 API Key：访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)

**使用 MiniMax API：**

```toml
default_provider = "custom:https://api.minimaxi.com/v1"
default_model = "abab6.5s-chat"
api_key = "your-minimax-api-key"
```

支持的 MiniMax 模型：
- `abab6.5s-chat` - MiniMax ABAB6.5S 对话模型
- `abab6.5g-chat` - MiniMax ABAB6.5G 对话模型

**使用 GiteeAI（免费模型）：**

```toml
default_provider = "custom:https://ai.gitee.com/v1"
default_model = "GLM-4.7-Flash"
api_key = "your-gitee-ai-api-key"
```

**使用 OpenAI：**

```toml
default_provider = "openai"
default_model = "gpt-4o"
api_key = "your-openai-api-key"
```

**使用 Anthropic Claude：**

```toml
default_provider = "anthropic"
default_model = "claude-sonnet-4-20250514"
api_key = "your-anthropic-api-key"
```

**使用 Google Gemini：**

```toml
default_provider = "gemini"
default_model = "gemini-2.0-flash"
api_key = "your-google-api-key"
```

**使用 Ollama 本地模型：**

```toml
default_provider = "ollama"
default_model = "llama3.1:8b"
ollama_host = "http://localhost:11434"
```

**使用企业微信机器人：**

```toml
[channels.wecom]
enabled = true
corp_id = "your-corp-id"
agent_id = "your-agent-id"
secret = "your-secret"
```

获取企业微信应用凭证：
1. 登录企业微信管理后台：https://work.weixin.qq.com/
2. 进入「应用管理」→「应用」
3. 创建或选择自建应用
4. 在应用详情页获取 `agent_id` 和 `secret`
5. 在「接收消息」设置中配置「接收消息服务器的回调 URL」和「Token」

**使用企业微信机器人（Webhook 方式）：**

```toml
[channels.wecom]
enabled = true
bot_id = "your-bot-id"
bot_secret = "your-bot-secret"
allowed_users = ["*"]
```

**使用钉钉：**

```toml
[channels.dingtalk]
enabled = true
app_key = "your-app-key"
app_secret = "your-app-secret"
```

**使用飞书：**

```toml
[channels.lark]
enabled = true
app_id = "your-app-id"
app_secret = "your-app-secret"
```

**使用 Telegram：**

```toml
[channels.telegram]
enabled = true
bot_token = "your-bot-token"
```

**使用 Discord：**

```toml
[channels.discord]
enabled = true
bot_token = "your-bot-token"
```

**使用 Slack：**

```toml
[channels.slack]
enabled = true
bot_token = "xoxb-your-bot-token"
```

**使用 WhatsApp：**

```toml
[channels.whatsapp]
enabled = true
phone_id = "your-phone-id"
api_key = "your-api-key"
```

**使用 Signal：**

```toml
[channels.signal]
enabled = true
phone_number = "+1234567890"
```

**使用 iMessage：**

```toml
[channels.imessage]
enabled = true
```

**使用微信 iLink Bot（个人微信机器人）：**

```toml
[channels_config.weixin]
enable = true
```

**微信 iLink Bot 使用方法：**
1. 首次使用需要扫码登录：
   ```bash
   goclaw channel login
   ```
2. 扫描终端显示的二维码完成登录
3. 登录成功后，凭证会自动保存到 `~/.goclaw/weixin/accounts/` 目录
4. 之后启动 daemon 会自动加载已登录的账号：
   ```bash
   goclaw daemon
   ```
5. 在微信中给机器人发消息即可与 AI 对话

**特点：**
- 零配置：扫码登录后自动保存凭证，无需手动配置 token
- 多账号支持：可登录多个微信账号
- 自动重连：服务重启后自动加载已保存的账号

**注意**：配置文件位置已更改为 `~/.goclaw/config.toml`，不再使用 `~/.goclaw/config.toml`

### 技能系统

GoClaw 支持多种技能类型：
- **Node.js 技能**：使用 JavaScript/TypeScript 编写
- **Python 技能**：使用 Python 编写
- **WASM 技能**：使用 WebAssembly 编写

#### 技能目录

技能目录位于 `~/.goclaw/workspace/skills/`，结构如下：

```
~/.goclaw/workspace/skills/
├── hello-world-skill/
│   ├── skill.json      # 技能配置
│   └── index.js        # 技能入口
├── stock-analyzer-skill/
│   ├── SKILL.toml      # 技能配置
│   ├── index.js        # 技能入口
│   └── scripts/        # Python 脚本
└── email-sender-skill/
    ├── SKILL.toml
    ├── index.js
    └── package.json
```

#### 使用 SkillForge 自动发现技能

GoClaw 内置 SkillForge 功能，可以自动从 GitHub 发现和集成技能：

```toml
[skillforge]
enabled = true
github_token = "your-github-token"
```

运行技能发现：
```bash
goclaw skills discover
```

### 语音输入支持

GoClaw 支持语音消息输入：
- **微信/企业微信**：自动使用微信内置语音识别
- **Whisper API**：支持 Groq/OpenAI Whisper 进行语音转文字

#### 配置 Whisper 转录

```toml
[transcription]
enabled = true
api_url = "https://api.groq.com/openai/v1/audio/transcriptions"
model = "whisper-large-v3-turbo"
api_key = "your-groq-api-key"
```

### 语音唤醒

GoClaw 支持语音唤醒 AI 助手：

```toml
[voice.wakeword]
enabled = true
wake_word = "hey claw"
sensitivity = 0.5
timeout_secs = 30
```

### RAG 知识库

GoClaw 支持 RAG（检索增强生成）功能，可以加载硬件手册等技术文档：

```toml
[rag]
enabled = true
workspace_dir = "~/.goclaw/workspace"
datasheet_dir = "datasheets"
```

### Verifiable Intent 安全凭证

GoClaw 实现企业级 Verifiable Intent 安全系统：

```toml
[verifiable_intent]
enabled = true
```

### 企业工具集成

GoClaw 支持多种企业工具集成：

#### Jira

```toml
[integrations.jira]
enabled = true
url = "https://your-domain.atlassian.net"
email = "your-email@example.com"
api_token = "your-api-token"
```

#### Notion

```toml
[integrations.notion]
enabled = true
api_key = "your-notion-api-key"
```

#### Microsoft 365

```toml
[integrations.microsoft365]
enabled = true
tenant_id = "your-tenant-id"
client_id = "your-client-id"
client_secret = "your-client-secret"
```

#### Google Workspace

```toml
[integrations.google]
enabled = true
credentials_file = "path/to/credentials.json"
```

### 分布式架构

GoClaw 支持分布式部署：

#### 节点注册

```toml
[nodes]
enabled = true
node_id = "node-1"
registry = "consul://localhost:8500"
```

#### 隧道连接

```toml
[tunnels]
enabled = true
server = "tunnel.example.com"
token = "your-tunnel-token"
```

### 可观测性

GoClaw 内置完整的可观测性功能：

#### OpenTelemetry

```toml
[observability.otel]
enabled = true
endpoint = "localhost:4317"
service_name = "goclaw"
```

#### Prometheus 指标

```toml
[observability.prometheus]
enabled = true
port = 9090
```

#### 分布式追踪

```toml
[observability.tracing]
enabled = true
sampler_type = "probabilistic"
sampler_param = 0.1
```

## 🚀 使用方法

### 启动 Daemon

```bash
./bin/goclaw daemon
```

### HTTP API 交互

GoClaw 提供 HTTP API，可以通过 HTTP 请求与 AI 助手交互：

```bash
# 发送消息
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "你好"}'

# 分析股票
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "帮我分析一下贵州茅台的股票"}'

# 发送邮件
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "请发送邮件到 270901361@qq.com，主题是\"Hello GoClaw\"，内容是\"测试内容\"}'
```

### Web 界面

访问 `http://localhost:4096` 使用 Web 界面进行交互。

## 🎯 功能说明

### 1. 邮件发送技能

在消息框中输入：
```
请发送邮件到 270901361@qq.com，主题是"Hello GoClaw"，内容是"测试内容"
```

### 2. 股票分析技能

支持 A股、港股、美股分析：

```
# 按股票名称
分析贵州茅台
分析腾讯控股
分析苹果AAPL

# 按股票代码
分析 600519
分析 00700
分析 AAPL

# 触发关键词
股票推荐
股票买卖点
A股分析
港股分析
美股分析
```

### 3. 多技能协同执行

GoClaw 支持多技能自动协同执行，Agent 可以智能识别用户需求并自动调用多个技能完成任务。

#### 示例：股票分析 + 邮件发送

```
帮我分析一下爱尔眼科这个股票，把结果发到我的邮箱
```

Agent 会自动执行以下步骤：
1. 使用股票分析技能获取爱尔眼科的详细数据
2. 从记忆体中搜索并获取用户的邮箱地址
3. 使用邮件发送技能将分析报告发送到邮箱

#### 记忆体功能

GoClaw 内置智能记忆体系统，支持：
- **自动存储**：对话内容自动保存到 SQLite 记忆体
- **智能检索**：基于 FTS5 全文搜索，快速检索相关信息
- **上下文关联**：Agent 可以根据上下文自动查询相关记忆

#### 存储邮箱地址

通过 HTTP API 存储邮箱地址：

```bash
curl -X POST http://localhost:4096/api/memory \
  -H "Content-Type: application/json" \
  -d '{"key":"my_email","content":"email:270901361@qq.com","category":"context"}'
```

#### 查询记忆体

```bash
# 查询所有记忆体
curl http://localhost:4096/api/memory

# 删除特定记忆体
curl -X DELETE http://localhost:4096/api/memory/my_email
```

#### 智能记忆体查询

当用户提到"邮箱"、"邮件"等关键词时，Agent 会自动使用 `memory_recall` 工具搜索记忆体中的邮箱地址，无需用户重复提供。

### 4. 记忆体 API

#### 添加记忆体

```bash
curl -X POST http://localhost:4096/api/memory \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user_preference",
    "content": "我喜欢技术类股票",
    "category": "preference"
  }'
```

#### 查询记忆体

```bash
# 获取所有记忆体
curl http://localhost:4096/api/memory

# 返回格式
{
  "count": 2,
  "entries": [
    {
      "id": "my_email",
      "key": "my_email",
      "content": "email:270901361@qq.com",
      "category": "context",
      "created_at": "2026-03-05T01:08:17.071233Z",
      "updated_at": "2026-03-05T01:15:05.344436Z"
    }
  ]
}
```

#### 删除记忆体

```bash
curl -X DELETE http://localhost:4096/api/memory/my_email
```

## 📸 效果展示

### Web 界面

![goclaw01](https://github.com/user-attachments/assets/8925e214-081a-4e54-a026-be870bd1585b)
![goclaw02](https://github.com/user-attachments/assets/7c87fdd8-4218-4c9c-b436-149022cc11ee)
![goclaw03](https://github.com/user-attachments/assets/b9aed7d3-148f-4655-bc65-4b5ff658a425)
![goclaw04](https://github.com/user-attachments/assets/4b823e1d-f01d-403c-87f5-30fd833be4d6)
![goclaw05](https://github.com/user-attachments/assets/0b58e49a-5453-45fa-9c8d-4aaf8938d440)
![goclaw06](https://github.com/user-attachments/assets/d59e9da5-99ae-488e-933c-a1f2ed76de41)

### 股票分析

![goclaw-st01](https://github.com/user-attachments/assets/d8b6626c-78dd-4602-8d46-01a008b36ee3)
![goclaw-st01](https://github.com/user-attachments/assets/443bea95-01a1-4005-93d6-7ec499c71524)

### 企业微信

![goclaw-we01](https://github.com/user-attachments/assets/857f0790-1a3a-4d68-8c0b-8643715701c1)

### 飞书
![goclaw-we02](https://github.com/user-attachments/assets/79dcb133-b3b3-40dc-a943-d39c7f6223d3)

## 🛠️ 技术栈

- **后端**：Go 1.21+
- **前端**：Vue 3 + Tailwind CSS
- **数据库**：SQLite、Qdrant
- **技能**：Node.js、Python
- **API**：RESTful API + WebSocket

## 📝 开发计划

- [ ] 完善所有渠道的测试
- [ ] 增加更多技能
- [ ] 优化 Web 界面
- [ ] 添加用户认证
- [ ] 支持多用户

## 📝 更新日志

### 2026-04-07 - v1.3.0

**新增功能**:
- ✨ 新增 15+ 渠道支持：Telegram、Discord、Slack、Signal、iMessage、Matrix、IRC、WhatsApp、Bluesky、Nostr、QQ、Reddit、LinkedIn、Twitter 等
- ✨ 新增 Hands 多智能体系统，支持多智能体协同工作
- ✨ 新增 SkillForge 技能工厂，自动发现和集成 GitHub 技能
- ✨ 新增 RAG 检索增强，支持硬件手册知识库
- ✨ 新增 Verifiable Intent 企业级安全凭证系统
- ✨ 新增语音唤醒功能，支持 "hey claw" 激活
- ✨ 新增分布式架构，支持多节点部署和隧道连接
- ✨ 新增 WASM 插件系统
- ✨ 新增 MCP (Model Context Protocol) 集成
- ✨ 新增完整可观测性支持：OpenTelemetry、Prometheus、分布式追踪
- ✨ 新增企业工具集成：Jira、Notion、Microsoft 365、Google Workspace
- ✨ 新增硬件支持：Serial、GPIO、Firmware 刷写
- ✨ 新增成本追踪功能
- ✨ 新增 Canvas 可视化功能

**使用方法**:
```bash
# 启动 GoClaw
goclaw daemon

# 启用分布式节点
goclaw node join

# 启用隧道
goclaw tunnel create

# 启用技能发现
goclaw skills discover
```

### 2026-03-24 - v1.2.0

**新增功能**:
- ✨ 添加微信 iLink Bot 渠道支持，实现个人微信机器人功能
- ✨ 支持扫码登录，零配置使用
- ✨ 自动保存和加载登录凭证
- ✨ 支持多微信账号管理

**使用方法**:
```bash
# 扫码登录微信
goclaw channel login

# 启动服务
goclaw daemon
```

### 2026-03-09 - v1.1.0

**新增功能**:
- ✨ 添加流式卡片回复功能，飞书/钉钉支持实时展示 AI 生成过程
- ✨ 流式卡片显示用户问题，提升交互体验
- ✨ 支持消息去重，避免重复处理相同消息

**改进**:
- 📝 优化消息处理流程
- 📝 增强错误处理和日志记录

**技术实现**:
- 使用飞书 Card Kit Streaming API 实现流式卡片
- 支持实时更新卡片内容，最大 10 次/秒
- 自动管理 token 缓存和权限验证

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) - 原始项目
- [Vue.js](https://vuejs.org/) - 前端框架
- [Go](https://golang.org/) - 后端语言
- [SQLite](https://sqlite.org/) - 嵌入式数据库
- [Qdrant](https://qdrant.tech/) - 向量数据库

## 📞 联系方式

- GitHub: [lj7788](https://github.com/lj7788)
- Email: lj7788@126.com

---

⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！
