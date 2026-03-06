# 配置说明

## 配置文件位置

配置文件位于 `~/.goclaw/config.yaml`

## 配置项

### 认证配置

```yaml
auth:
  enable_wechat_login: true  # 是否启用微信登录
  enable_admin_login: true   # 是否启用管理员登录
  admin_username: "admin"    # 管理员用户名
  admin_password: "admin"    # 管理员密码
```

### 网关配置

```yaml
gateway:
  port: 4096                 # 网关端口
  host: "0.0.0.0"            # 网关主机
```

### WebSocket配置

```yaml
websocket:
  enabled: true              # 是否启用WebSocket
  path: "/api/ws/chat"       # WebSocket路径
  max_connections: 100       # 最大连接数
```

### 数据库配置

```yaml
database:
  type: "sqlite"             # 数据库类型
  path: "~/.goclaw/db.sqlite" # 数据库路径
```

### 日志配置

```yaml
log:
  level: "info"              # 日志级别
  file: "~/.goclaw/logs/app.log" # 日志文件路径
```

## 配置示例

```yaml
auth:
  enable_wechat_login: true
  enable_admin_login: true
  admin_username: "admin"
  admin_password: "admin"

gateway:
  port: 4096
  host: "0.0.0.0"

websocket:
  enabled: true
  path: "/api/ws/chat"
  max_connections: 100

database:
  type: "sqlite"
  path: "~/.goclaw/db.sqlite"

log:
  level: "info"
  file: "~/.goclaw/logs/app.log"
```

## 配置说明

### 认证配置

- `enable_wechat_login`: 是否启用微信登录，默认为true
- `enable_admin_login`: 是否启用管理员登录，默认为true
- `admin_username`: 管理员用户名，默认为"admin"
- `admin_password`: 管理员密码，默认为"admin"

### 网关配置

- `port`: 网关端口，默认为4096
- `host`: 网关主机，默认为"0.0.0.0"

### WebSocket配置

- `enabled`: 是否启用WebSocket，默认为true
- `path`: WebSocket路径，默认为"/api/ws/chat"
- `max_connections`: 最大连接数，默认为100

### 数据库配置

- `type`: 数据库类型，默认为"sqlite"
- `path`: 数据库路径，默认为"~/.goclaw/db.sqlite"

### 日志配置

- `level`: 日志级别，默认为"info"
- `file`: 日志文件路径，默认为"~/.goclaw/logs/app.log"

## 配置修改

修改配置文件后，需要重启服务才能生效。

```bash
./goclaw daemon
```