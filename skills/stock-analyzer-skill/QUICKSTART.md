# GoClaw Stock Analyzer Skill 快速使用指南

## 📚 概述

`stock-analyzer-skill` 已成功集成到 GoClaw 中，现在可以通过 HTTP API 使用股票分析功能。

## 🚀 快速开始

### 1. 启动 GoClaw Daemon

```bash
cd /Users/haha/.zeroclaw/GoClaw
./bin/goclaw daemon
```

### 2. 发送股票分析请求

在另一个终端执行：

```bash
# 分析 A股 - 贵州茅台
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "帮我分析一下贵州茅台的股票"}'

# 分析 港股 - 腾讯控股
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "分析腾讯控股"}'

# 分析 美股 - 苹果
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "分析苹果AAPL"}'

# 分析 美股 - 美光
curl -X POST http://localhost:4096/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "分析美光MU"}'
```

## 🎯 支持的查询方式

### 按股票名称
```
分析贵州茅台
分析腾讯控股
分析苹果AAPL
分析美光MU
分析特斯拉TSLA
分析英伟达NVDA
```

### 按股票代码
```
分析 600519
分析 00700
分析 AAPL
分析 MU
```

### 触发关键词
以下关键词会自动触发股票分析：

**通用关键词：**
- 分析股票
- 股票推荐
- 股票买卖点
- 股票研究

**市场关键词：**
- A股分析、港股分析、美股分析
- 恒生指数、纳斯达克、标普500、道琼斯
- 中概股

**股票名称：**
- 贵州茅台、腾讯、美团、小米、比亚迪股份
- 苹果AAPL、特斯拉TSLA、英伟达NVDA、美光MU
- 谷歌GOOGL、微软MSFT、亚马逊AMZN、Meta
- 阿里巴巴BABA、拼多多PDD、京东JD

## 📊 输出示例

```
═══════════════════════════════════════════════════════════════
  📉 贵州茅台 (600519)
═════════════════════════════════════════════════════════════════

💰 当前价格: 1440.11
📉 涨跌额:   -14.91
📉 涨跌幅:   -1.02%

────────────────────────────────────────────────────────────────────

📊 基本信息
  今开: 1450.00
  昨收: 1455.02
  最高: 1457.00
  最低: 1436.66
  涨停: 1600.52
  跌停: 1309.52

────────────────────────────────────────────────────────────────────

💼 市场数据
  成交额: 51.15亿
  总市值: 1.80万亿
  流通市值: 1.80万亿
  换手率: 0.28%
  量比: 0.96

────────────────────────────────────────────────────────────────────

📈 估值指标
  市盈率(PE): 20.93
  市净率(PB): 7.94

────────────────────────────────────────────────────────────────────

💡 综合分析
  基本面评分: 5/10
  基本面中性
  
  资金面评分: 5/10
  资金面中性
  
  综合评分: 10/20
  ✅ 投资建议: 推荐买入

────────────────────────────────────────────────────────────────────

🎯 买卖价位建议
  建仓价位: 1396.91
  目标价位: 1584.12
  止损价位: 1324.90

═════════════════════════════════════════════════════════════════
```

## 🏗️ 技术架构

```
用户请求 (HTTP API)
    ↓
GoClaw Agent
    ↓
StockAnalyzerTool (Go)
    ↓
index.js (Node.js)
    ↓
fetch_stock.py (Python)
    ↓
东方财富 API
    ↓
分析报告 (JSON/Markdown)
    ↓
返回给用户
```

## 📁 文件结构

```
/Users/haha/.zeroclaw/workspace/skills/stock-analyzer-skill/
├── index.js              # Node.js 入口文件
├── package.json          # Node.js 配置
├── skill.json            # 技能定义
├── _meta.json            # 元数据
├── SKILL.md              # 详细文档
├── README.md             # 使用指南
├── scripts/
│   ├── fetch_stock.py    # Python 数据抓取脚本
│   └── requirements.txt  # Python 依赖
├── assets/
│   ├── report_template.html  # HTML 报告模板
│   └── report_template.md   # Markdown 报告模板
└── references/
    └── eastmoney_guide.md   # 东方财富指南
```

## 🔧 集成到 GoClaw

已自动集成到 GoClaw 的以下位置：

1. **工具定义**：`pkg/tools/stock_analyzer_tool.go`
2. **主程序**：`cmd/main.go` (3处)
   - daemon 模式
   - run 模式
   - chat 模式

## ✅ 测试验证

```bash
# 测试 A股
cd /Users/haha/.zeroclaw/workspace/skills/stock-analyzer-skill
node index.js --stock 600519 --market sh

# 测试 港股
node index.js --stock 00700 --market hk

# 测试 美股
node index.js --stock AAPL --market us
```

## 📝 注意事项

1. **数据来源**：东方财富网，仅供参考，不构成投资建议
2. **登录建议**：建议登录东方财富网账号以获取完整数据
3. **请求限制**：避免短时间内频繁查询
4. **交易时间**：仅在交易时间内能获取实时数据
5. **网络要求**：需要稳定的网络连接

## 🐛 故障排除

### 问题：无法获取股票数据
- 检查网络连接
- 确认股票代码是否正确
- 尝试登录东方财富网账号

### 问题：Python 脚本执行失败
```bash
# 检查 Python 版本
python3 --version

# 安装依赖
cd /Users/haha/.zeroclaw/workspace/skills/stock-analyzer-skill/scripts
pip install -r requirements.txt
```

### 问题：GoClaw 构建失败
```bash
cd /Users/haha/.zeroclaw/GoClaw
go mod tidy
go build -o bin/goclaw cmd/main.go
```

## 🎓 更多信息

- 详细文档：`/Users/haha/.zeroclaw/workspace/skills/stock-analyzer-skill/SKILL.md`
- 使用指南：`/Users/haha/.zeroclaw/workspace/skills/stock-analyzer-skill/README.md`
- 东方财富指南：`/Users/haha/.zeroclaw/workspace/skills/stock-analyzer-skill/references/eastmoney_guide.md`

## 📞 支持

如有问题，请检查：
1. GoClaw 日志输出
2. 技能日志（`[INFO]` 和 `[ERROR]` 标记）
3. 东方财富网站是否可访问

---

**享受股票分析！** 📈💰
