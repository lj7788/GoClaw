# GoClaw Nodes 模块实现计划

## 概述

Nodes 模块用于将配套设备（macOS/iOS/Android/headless）连接到 GoClaw Gateway，暴露本地能力如 camera、screen、canvas、system 等。

## 实现步骤

### Phase 1: 核心架构（基础）

#### 1.1 定义 Nodes 数据结构
- 创建 `pkg/nodes/` 目录
- 定义 `Node` 结构体：ID、Name、Role、Capabilities、Status、LastSeen
- 定义 `NodeCapability` 枚举：camera、screen、canvas、system、notifications、device
- 定义连接协议：WebSocket 消息格式

#### 1.2 节点注册与发现
- 实现节点连接握手协议
- 设备配对/审批流程
- 节点状态管理（在线/离线）

#### 1.3 Gateway 集成
- 在 Gateway 添加 `/ws/node` 路由
- 节点认证与授权
- 节点心跳保活

### Phase 2: 节点能力接口

#### 2.1 Screen 截图
- 节点端截图能力注册
- 远程截图请求/响应
- 图片压缩与传输

#### 2.2 Camera 摄像头
- 摄像头权限处理
- 视频流/图片捕获

#### 2.3 Canvas 画布
- 画布操作命令
- 绘图指令传输

#### 2.4 System 命令执行
- 远程 shell 命令执行
- 命令结果返回

### Phase 3: 安全与权限

#### 3.1 配对审批
- 首次连接需要审批
- 节点身份验证

#### 3.2 权限控制
- 按节点分配能力
- 命令执行审批（可复用现有 approval 模块）

### Phase 4: CLI 工具

#### 4.1 节点管理命令
```
goclaw nodes list          # 列出节点
goclaw nodes approve <id>  # 审批节点
goclaw nodes reject <id>   # 拒绝节点
goclaw nodes describe <id> # 节点详情
goclaw nodes remove <id>   # 移除节点
```

### Phase 5: 节点端 SDK（可选）

#### 5.1 轻量级节点程序
- 提供 `goclaw node run` 命令
- 自动发现 Gateway
- 能力注册

## 注意事项

### 1. 协议兼容性
- 尽量与 OpenClaw 协议兼容，便于后续互通
- 使用 JSON 格式的 WebSocket 消息

### 2. 安全性
- 节点连接需要认证 token
- 敏感操作需要用户审批
- 防止节点被恶意控制

### 3. 性能
- 大文件（截图）需要压缩
- 心跳间隔合理设置
- 连接超时处理

### 4. 错误处理
- 节点断线重连机制
- 命令超时处理
- 优雅降级

### 5. 现有代码复用
- 复用 `pkg/gateway/websocket.go` 的 WebSocket 实现
- 复用 `pkg/approval` 的审批流程
- 复用 `pkg/tools` 的命令执行逻辑

## 预计工作量

| 模块 | 工作量 |
|------|--------|
| 核心架构 | 2-3 天 |
| Screen 截图 | 1-2 天 |
| Camera 摄像头 | 1-2 天 |
| System 命令 | 1-2 天 |
| 安全与权限 | 1-2 天 |
| CLI 工具 | 1 天 |
| 测试与调试 | 2-3 天 |
| **总计** | **9-16 天** |

## 依赖

- `github.com/gorilla/websocket` - 已使用
- 截图库（平台相关）
- 摄像头捕获（平台相关）
