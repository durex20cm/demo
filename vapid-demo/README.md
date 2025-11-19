# VAPID Web Push 通知完整示例

这是一个完整的 VAPID (Voluntary Application Server Identification) Web Push 通知演示项目，展示了从密钥生成到消息推送的完整流程。

## 功能特性

- ✅ **密钥生成**：使用 Go 生成 VAPID 密钥对
- ✅ **前端订阅**：浏览器端订阅推送通知
- ✅ **后端推送**：使用 Golang 实现推送服务器
- ✅ **消息接收**：Service Worker 接收并显示通知

## 项目结构

```
vapid-demo/
├── main.go              # Go 后端服务器
├── cmd/
│   └── generate-keys.go # VAPID 密钥生成工具
├── go.mod              # Go 模块依赖
├── Makefile            # Make 命令文件
├── static/
│   ├── index.html      # 前端页面
│   └── sw.js           # Service Worker
└── README.md           # 项目说明
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

或者使用 Makefile：

```bash
make deps
```

### 2. 生成 VAPID 密钥

运行密钥生成工具：

```bash
go run cmd/generate-keys.go
```

或者使用 Makefile：

```bash
make keys
```

输出示例：
```
=== VAPID 密钥生成成功 ===

请将以下密钥添加到环境变量中：

export VAPID_PUBLIC_KEY=BHx...
export VAPID_PRIVATE_KEY=xyz...

或者创建 .env 文件（如果使用环境变量加载工具）：

VAPID_PUBLIC_KEY=BHx...
VAPID_PRIVATE_KEY=xyz...
```

#### 使用 `web-push` 工具生成 VAPID 密钥（可选）

除了 Go 版本的生成工具，还可以用 `web-push`（Node.js 跨平台工具）快速生成密钥：

```bash
npm install -g web-push
web-push generate-vapid-keys
```

命令行会输出类似：

```
=======================================
Public Key:
BHx...

Private Key:
xyz...
=======================================
```

将输出的 **Public Key** 和 **Private Key** 配置到你的环境变量中即可。


### 3. 设置环境变量

有两种方式设置环境变量：

#### 方式一：使用 .env 文件（推荐）

创建 `.env` 文件并添加密钥：

```bash
# 创建 .env 文件
cat > .env << EOF
VAPID_PUBLIC_KEY=你的公钥
VAPID_PRIVATE_KEY=你的私钥
PORT=8080
EOF
```

#### 方式二：使用系统环境变量

```bash
export VAPID_PUBLIC_KEY="你的公钥"
export VAPID_PRIVATE_KEY="你的私钥"
```

**注意**：如果同时存在 `.env` 文件和系统环境变量，系统环境变量会优先使用。

### 4. 启动服务器

```bash
go run main.go
```

或者使用 Makefile：

```bash
make run
```

服务器将在 `http://localhost:8080` 启动。

### 5. 访问前端页面

在浏览器中打开 `http://localhost:8080`，然后：

1. **订阅推送通知**：点击"订阅推送通知"按钮
2. **允许通知权限**：浏览器会请求通知权限，请点击"允许"
3. **发送测试通知**：填写通知标题和内容，点击"发送测试通知"

## Makefile 命令

项目提供了 Makefile 来简化常用操作：

| 命令 | 说明 |
|------|------|
| `make deps` | 下载并整理 Go 模块依赖 |
| `make keys` | 生成 VAPID 密钥对 |
| `make run` | 启动开发服务器（需要先设置环境变量） |
| `make build` | 构建可执行文件 `vapid-demo` |
| `make clean` | 清理构建文件和缓存 |

使用示例：

```bash
# 安装依赖
make deps

# 生成密钥
make keys

# 启动服务器（需要先设置 VAPID 密钥环境变量）
make run

# 构建生产版本
make build

# 清理构建文件
make clean
```

## 使用说明

### 前端流程

1. **获取 VAPID 公钥**：前端从 `/api/vapid-public-key` 获取公钥
2. **注册 Service Worker**：自动注册 `sw.js` 处理推送消息
3. **订阅推送**：使用 Push API 订阅，并将订阅信息发送到 `/api/subscribe`
4. **接收通知**：Service Worker 监听 `push` 事件并显示通知

### 后端流程

1. **接收订阅**：`/api/subscribe` 接收并存储订阅信息
2. **发送推送**：`/api/push` 接收推送请求，向所有订阅者发送通知
3. **处理失效订阅**：自动清理已失效的订阅（HTTP 410）

## API 接口

### GET /api/vapid-public-key

获取 VAPID 公钥，用于前端订阅。

**响应：**
```json
{
  "publicKey": "BHx..."
}
```

### POST /api/subscribe

接收前端订阅信息。

**请求体：**
```json
{
  "endpoint": "https://fcm.googleapis.com/...",
  "keys": {
    "p256dh": "base64编码的密钥",
    "auth": "base64编码的认证密钥"
  }
}
```

**响应：**
```json
{
  "status": "success",
  "message": "订阅成功"
}
```

### POST /api/push

发送推送通知到所有订阅者。

**请求体：**
```json
{
  "title": "通知标题",
  "body": "通知内容",
  "icon": "/static/icon.png",
  "url": "https://example.com"
}
```

**响应：**
```json
{
  "status": "success",
  "success": 1,
  "failed": 0,
  "total": 1,
  "message": "已推送 1 条消息，成功 1，失败 0"
}
```

## 技术栈

- **后端**：Go 1.21+
- **Web Push 库**：github.com/SherClockHolmes/webpush-go
- **路由**：github.com/gorilla/mux
- **前端**：原生 JavaScript + Service Worker API

## 注意事项

1. **HTTPS 要求**：Web Push API 要求使用 HTTPS（localhost 除外）
2. **浏览器支持**：需要支持 Service Worker 和 Push API 的现代浏览器
3. **通知权限**：用户必须授予通知权限才能接收推送
4. **VAPID 密钥**：生产环境请妥善保管私钥，不要提交到代码仓库
5. **图标文件**：通知图标（`/static/icon.png`）是可选的，如果不存在，通知将不显示图标

## 开发建议

- 生产环境建议使用数据库存储订阅信息，而不是内存
- 可以添加用户认证，实现个性化推送
- 可以添加推送队列，处理大量推送请求
- 建议添加推送日志和统计功能

## 许可证

MIT License

