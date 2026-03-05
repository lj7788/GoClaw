# GoClaw

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
- 🔌 **多渠道集成**：支持钉钉、微信、Telegram、Slack 等多种通知渠道
- 🎨 **现代化 Web 界面**：使用 Vue 3 重写，支持中文界面
- 🛠️ **丰富的工具集**：内置文件操作、Web 搜索、Git 操作等工具

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
[provider]
name = "bailian"
model = "qwen-plus"
api_key = "your-bailian-api-key"
```

支持的百炼模型：
- `qwen-plus` - 通义千问 Plus 模型
- `qwen-coder-plus` - 通义千问 Coder Plus 模型（适合编程）
- `qwen-coder-turbo` - 通义千问 Coder Turbo 模型（快速编程）
- `qwen-max` - 通义千问 Max 模型（最强性能）
- `qwen-turbo` - 通义千问 Turbo 模型（快速响应）
- `qwen-flash` - 通义千问 Flash 模型（极速响应）

获取 API Key：访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)

**使用 GiteeAI（免费模型）：**

```toml
[provider]
name = "gitee"
model = "GLM-4.7-Flash"
url = "custom:https://ai.gitee.com/v1"
api_key = "your-gitee-ai-api-key"
```

**使用 OpenAI：**

```toml
[provider]
name = "openai"
model = "gpt-4"
api_key = "your-openai-api-key"
```

**注意**：配置文件位置已更改为 `~/.goclaw/config.toml`，不再使用 `~/.goclaw/config.toml`

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
- 所有贡献者

## 📞 联系方式

- GitHub: [lj7788](https://github.com/lj7788)
- Email: lj7788@126.com

---

⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！
