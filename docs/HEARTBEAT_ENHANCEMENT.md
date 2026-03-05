# GoClaw Heartbeat 模块增强计划

## 当前 GoClaw Heartbeat 现状

现有 `pkg/heartbeat/engine.go` 提供了基础的事件引擎：
- 简单的 tick 事件触发机制
- 基本的 Handler 注册/注销
- 心跳 payload 结构（Timestamp、Uptime、Status）

## OpenClaw Heartbeat 功能

根据 OpenClaw 文档，heartbeat 是"周期性感知"机制，核心特点：

### 1. 周期性 Agent 调用
- 在固定间隔（默认30分钟）触发 Agent 运行
- 让 Agent 检查需要关注的事项，避免频繁打扰用户
- Agent 返回 `HEARTBEAT_OK` 表示无操作

### 2. HEARTBEAT.md 工作清单
- 从 workspace 读取 `HEARTBEAT.md` 作为检查清单
- 保持简短以节省 token

### 3. 配置选项
- `every`: 心跳间隔（如 "30m", "1h"）
- `target`: 消息发送目标
- `activeHours`: 活动时间限制（local time）

### 4. 与 Cron 的区别
- **Heartbeat**: 周期性检查，批量处理小任务
- **Cron**: 精确时间执行（如每天9:00）

---

## 需要增强的功能

### Phase 1: 核心增强（必须）

#### 1.1 Agent 集成
- [ ] 心跳触发时调用 Agent 进行检查
- [ ] 支持读取 `HEARTBEAT.md` 文件作为上下文
- [ ] 解析 Agent 返回的 `HEARTBEAT_OK` 标记
- [ ] 结果推送到对应渠道

#### 1.2 配置化
- [ ] 支持配置心跳间隔（30m, 1h 等）
- [ ] 支持活动时间限制（activeHours）
- [ ] 支持消息发送目标配置

#### 1.3 与 Workspace 集成
- [ ] 读取 `~/.goclaw/workspace/HEARTBEAT.md`
- [ ] 将清单内容注入 Agent 上下文

### Phase 2: 高级功能

#### 2.1 Rotating Heartbeat（轮转心跳）
- [ ] 单个心跳轮转多个检查任务
- [ ] 根据任务到期时间调度
- [ ] 避免"所有任务同时触发"问题

#### 2.2 成本优化
- [ ] 使用最便宜的模型执行心跳检查
- [ ] 发现问题时才触发完整 Agent

#### 2.3 消息去重/聚合
- [ ] 避免频繁推送打扰用户
- [ ] 聚合多个检查结果

### Phase 3: 与现有模块集成

#### 3.1 渠道集成
- [ ] 支持多渠道消息推送（与 channels 模块集成）
- [ ] 支持指定回复到哪个 session/channel

#### 3.2 与 Skills 集成
- [ ] 支持心跳触发特定 skill 执行

---

## 实现架构建议

```
pkg/heartbeat/
├── engine.go          # 现有引擎（保留）
├── agent.go           # 新增：Agent 触发逻辑
├── config.go          # 新增：配置结构
├── heartbeat.md       # 新增：Workspace 读取
├── scheduler.go       # 新增：轮转调度（可选）
└── reporter.go        # 新增：结果推送
```

---

## 预计工作量

| 功能 | 工作量 |
|------|--------|
| Agent 集成 | 1-2 天 |
| 配置化 | 0.5 天 |
| Workspace 集成 | 0.5 天 |
| 渠道推送 | 1 天 |
| Rotating Heartbeat | 2-3 天 |
| **总计** | **5-7 天** |
