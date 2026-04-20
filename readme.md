# LiveChat

一个基于 Go 的在线客服系统，当前形态是单体服务加 Vue 3 前端：

- Go 服务同时承载 HTTP、WebSocket、gRPC agent 调度
- 可选使用 Kafka 对 agent 调用做异步解耦
- 后台登录、安装页、客服工作台、访客聊天页统一由 `frontend/` 构建
- 构建产物输出到 `static/dist/`，由 Go 服务直接托管
- 支持 MySQL、Redis、Prometheus、OpenTelemetry、Jaeger

## 当前能力

### 客服侧

- 登录页：`/login`
- 工作台：`/main`
- 我的会话、待分配会话、最近访客、快捷回复、黑名单、个人资料统一处理
- 支持发送文本、图片、附件
- 支持转接会话、结束会话
- 支持客服直接接管待分配会话，并保留手工转接给其他客服
- 支持查看人工客服与 agent 状态、容量和待分配情况
- 支持为人工客服配置技能标签，按技能池参与自动分配

### 访客侧

- 访客聊天页：支持旧模式 `/livechat?user_id=<客服账号>`，也支持统一入口 `/livechat`、`/livechat?entry_id=<入口标识>`、`/livechat?service_line=<业务线>`
- WebSocket 实时收发消息
- 自动重连
- 图片上传、附件上传、截图粘贴、表情
- 暂无可接待客服时可进入待分配状态，保留会话与排队消息
- 待分配会话支持后台自动重试分配、超时扩散和长时间无活动自动回收
- 欢迎语、公告、历史消息分页加载

### 自动回复与 agent

- 自动回复支持 Redis 缓存
- Redis 不可用时自动退化到内存缓存
- 关键词支持精确匹配和最长包含匹配
- 人工客服离线且未命中自动回复时，可尝试分配给 gRPC agent
- agent 调用支持按配置切换到 Kafka 异步投递，缓冲 WebSocket 请求峰值与 agent 服务处理速率抖动

### 工程化能力

- `/healthz` 存活检查
- `/readyz` 就绪检查
- `/version` 版本信息
- `/metrics` Prometheus 指标
- 请求超时控制
- 访问限流
- gRPC agent 客户端熔断
- 优雅停机
- agent 调度链路状态可观测

## 技术栈

- 后端：Go、Gin、Gorm、Cobra、WebSocket、gRPC
- 前端：Vue 3、Vue Router、Vite
- 数据库：MySQL
- 缓存：Redis，可降级为内存缓存
- 消息队列：Kafka，可选
- 观测：Prometheus、OpenTelemetry、Jaeger

## 目录结构

```text
.
├─agent/                 agent 注册、分配、心跳与会话占用管理
├─agentpb/               gRPC 生成代码
├─cmd/                   CLI 与服务启动入口
├─common/                配置和全局常量
├─config/                MySQL、Redis 等配置
├─controller/            HTTP 控制器
├─frontend/              Vue 3 + Vite 前端源码
├─middleware/            超时、限流、日志、鉴权等中间件
├─models/                数据模型和数据库访问
├─proto/                 gRPC proto 定义
├─router/                页面路由与 API 路由
├─static/
│  ├─dist/               前端构建产物
│  ├─images/             图片、音频等静态资源
│  ├─js/                 嵌入式访客组件等保留脚本
│  └─templates/          说明目录，主页面模板已不再使用
├─tools/                 缓存、监控、tracing、工具函数
└─ws/                    WebSocket 会话与自动回复逻辑
```

## 运行要求

- Go 1.20+
- Node.js 18+
- MySQL 5.7+ 或兼容版本
- Redis 6+，可选

Redis 不是强依赖。Redis 不可用时，服务仍可启动，自动回复缓存会退化为内存实现。

## 快速开始

### 1. 创建数据库

```sql
CREATE DATABASE liveChat CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

### 2. 安装前端依赖

```bash
npm install
```

### 3. 构建前端

```bash
npm run build
```

构建完成后，产物会输出到 `static/dist/`。

### 4. 启动服务

```bash
go run . server
```

Windows：

```powershell
go run . server
```

### 5. 首次安装

首次启动且数据库表未初始化时，访问：

```text
http://127.0.0.1:8081/install
```

安装页会写入 `config/mysql.json` 并导入仓库根目录的 `import.sql`。

安装完成后可访问：

- 后台登录页：`http://127.0.0.1:8081/login`
- 后台工作台：`http://127.0.0.1:8081/main`
- 访客聊天页：`http://127.0.0.1:8081/livechat?user_id=agent`
- 智能分配入口：`http://127.0.0.1:8081/livechat`

如果数据库未初始化，访问 `/login`、`/main`、`/livechat` 会被重定向到 `/install`。

### 6. 命令行安装

如果你已经提前写好了 `config/mysql.json`，也可以直接执行：

```bash
go run . install
```

### 7. 默认账号

`import.sql` 会初始化一个默认客服账号：

- 账号：`agent`
- 密码：以你当前 `import.sql` 中的初始化数据为准；如果不确定，建议直接在登录页注册新账号使用

## 配置文件

### MySQL

编辑 `config/mysql.json`：

```json
{
  "Server": "127.0.0.1",
  "Port": "3306",
  "Database": "liveChat",
  "Username": "root",
  "Password": "your-password"
}
```

### Redis

编辑 `config/redis.json`：

```json
{
  "Host": "127.0.0.1",
  "Port": "6379",
  "Password": "",
  "DB": 0,
  "PoolSize": 10
}
```

## 页面入口

- `/install`：安装页
- `/login`：后台登录页
- `/main`：客服工作台
- `/livechat`：访客聊天页
- `/pannel`：兼容旧地址，重定向到 `/main`
- `/chat_main`：兼容旧地址，重定向到 `/main`
- `/setting`：兼容旧地址，重定向到 `/main?panel=profile`

## 关键接口

- `/check`：后台登录
- `/register`：后台注册
- `/install`：Web 安装
- `/visitor_login`：访客登录
- `/workbench/bootstrap`：工作台首屏数据
- `/ws_kefu`：客服 WebSocket
- `/ws_visitor`：访客 WebSocket
- `/uploadimg`：图片上传
- `/uploadfile`：附件上传
- `/agents/status`：agent 状态

## 系统端点

- `/healthz`：存活检查
- `/readyz`：就绪检查
- `/version`：版本、构建信息和 agent 状态
- `/metrics`：Prometheus 指标

## 启动参数

```bash
go run . server -p 8081 --grpc-port 9090
```

- `-p, --port`：HTTP 端口，默认 `8081`
- `--grpc-port`：gRPC 端口，默认 `9090`
- `-d, --daemon`：守护进程模式

## 环境变量

| 环境变量 | 默认值 | 说明 |
|---|---:|---|
| `LIVECHAT_SERVICE_NAME` | `goflylivechat` | 服务名 |
| `LIVECHAT_BUILD_VERSION` | 当前版本号 | 版本信息 |
| `LIVECHAT_BUILD_COMMIT` | `dev` | 构建提交号 |
| `LIVECHAT_BUILD_TIME` | 空 | 构建时间 |
| `LIVECHAT_HTTP_PORT` | `8081` | HTTP 端口 |
| `LIVECHAT_GRPC_PORT` | `9090` | gRPC 端口 |
| `LIVECHAT_HTTP_READ_TIMEOUT` | `10s` | HTTP 读超时 |
| `LIVECHAT_HTTP_WRITE_TIMEOUT` | `15s` | HTTP 写超时 |
| `LIVECHAT_HTTP_IDLE_TIMEOUT` | `60s` | HTTP 空闲超时 |
| `LIVECHAT_REQUEST_TIMEOUT` | `12s` | 请求总超时 |
| `LIVECHAT_SHUTDOWN_TIMEOUT` | `15s` | 优雅停机超时 |
| `LIVECHAT_RATE_LIMIT_RPS` | `8` | 请求限流速率 |
| `LIVECHAT_RATE_LIMIT_BURST` | `16` | 请求突发桶大小 |
| `LIVECHAT_KEFU_DEFAULT_MAX_SESSIONS` | `5` | 人工客服默认最大接待会话数 |
| `LIVECHAT_KEFU_DEFAULT_QUEUE` | `default` | 访客未命中显式客服时的默认队列名称 |
| `LIVECHAT_KEFU_PENDING_RETRY_INTERVAL` | `3s` | pending 会话后台自动重试分配的间隔 |
| `LIVECHAT_KEFU_PENDING_EXPAND_AFTER` | `10s` | pending 会话等待原客服/原队列超时后扩散到公共队列的阈值 |
| `LIVECHAT_KEFU_PENDING_TTL` | `10m` | pending 会话长时间无活动后自动回收的阈值 |
| `LIVECHAT_AGENT_HEARTBEAT_TTL` | `75s` | agent 心跳失效时间 |
| `LIVECHAT_AGENT_DIAL_TIMEOUT` | `2s` | agent gRPC 建连超时 |
| `LIVECHAT_AGENT_REQUEST_TIMEOUT` | `1200ms` | agent gRPC 请求超时 |
| `LIVECHAT_AGENT_DISPATCH_MODE` | `direct` | agent 调度模式，支持 `direct` 和 `kafka` |
| `LIVECHAT_AGENT_KAFKA_BROKERS` | `127.0.0.1:9092` | Kafka broker 列表，逗号分隔 |
| `LIVECHAT_AGENT_KAFKA_TOPIC` | `livechat.agent.session` | agent 调度事件 topic |
| `LIVECHAT_AGENT_KAFKA_GROUP_ID` | `livechat-agent-dispatcher` | agent 调度消费者组 |
| `LIVECHAT_AGENT_KAFKA_CLIENT_ID` | `livechat` | Kafka client id |
| `LIVECHAT_AGENT_KAFKA_DIAL_TIMEOUT` | `2s` | Kafka 建连超时 |
| `LIVECHAT_AGENT_KAFKA_READ_TIMEOUT` | `3s` | Kafka 读超时 |
| `LIVECHAT_AGENT_KAFKA_WRITE_TIMEOUT` | `3s` | Kafka 写超时 |
| `LIVECHAT_AGENT_KAFKA_ENQUEUE_TIMEOUT` | `800ms` | WebSocket 主路径投递 agent 事件的超时 |
| `LIVECHAT_AGENT_KAFKA_CONSUME_BACKOFF` | `1s` | Kafka 消费失败后的重试退避时间 |
| `LIVECHAT_BREAKER_TIMEOUT` | `5s` | 熔断恢复等待时间 |
| `LIVECHAT_BREAKER_HALF_OPEN_MAX` | `3` | 熔断半开最大请求数 |

## 客服技能池

人工客服技能当前通过每个客服自己的 `RoutingSkills` 配置保存，格式为逗号分隔字符串，例如：

```text
sales, support, refund
```

当前路由行为：

- 访客带 `service_line` 进入时，优先进入对应技能池
- 技能池内无可接待客服时，会进入 pending 队列
- 等待超过 `LIVECHAT_KEFU_PENDING_EXPAND_AFTER` 后，会扩散到公共池继续分配
- 技能标签可在工作台个人资料面板直接维护
- 工作台会展示队列、等待时长、未分配原因，并支持当前客服直接接管 pending 会话
- pending 会话在后台自动重试分配成功后，会通过实时事件同步更新工作台列表和当前会话详情
- 客服个人资料面板支持设置 `在线 / 暂离 / 繁忙` 以及“继续接收新会话”，路由时会据此决定是否参与分配
| `LIVECHAT_JAEGER_ENDPOINT` | `http://localhost:14268/api/traces` | Jaeger 上报地址 |
| `LIVECHAT_ENABLE_TRACING` | `false` | 是否启用 tracing |
| `LIVECHAT_ENABLE_METRICS` | `true` | 是否启用指标 |

## Agent 调度模式

默认情况下，访客无人值守场景下的 agent 分配与释放仍走进程内 gRPC 直连。

当请求速率和 agent 服务处理速率波动较大时，可以开启 Kafka 解耦：

```bash
LIVECHAT_AGENT_DISPATCH_MODE=kafka
LIVECHAT_AGENT_KAFKA_BROKERS=127.0.0.1:9092
go run . server
```

启用后：

- WebSocket 主路径只负责把 `assign` / `release` 事件写入 Kafka
- 后台消费者顺序消费同一访客的事件，并调用现有 gRPC agent 服务
- `readyz` 会额外检查 `agent_dispatch`
- `version` 会返回 `agent_dispatch_mode` 和 `agent_dispatch_ready`

当前实现仍基于单体内嵌 gRPC registry；如果未来需要多实例共享 agent 状态，需继续把 registry 从进程内状态演进为集中式存储或独立调度服务。

## 嵌入访客窗口

直接访问：

```text
http://127.0.0.1:8081/livechat?user_id=agent
```

脚本嵌入：

```html
<script>
  (function (global, document, baseUrl, callback) {
    const head = document.getElementsByTagName("head")[0];
    const script = document.createElement("script");
    script.type = "text/javascript";
    script.src = baseUrl + "/static/js/chat-widget.js";
    script.onload = script.onreadystatechange = function () {
      if (!this.readyState || this.readyState === "loaded" || this.readyState === "complete") {
        callback(baseUrl);
      }
    };
    head.appendChild(script);
  })(window, document, "http://127.0.0.1:8081", function (baseUrl) {
    CHAT_WIDGET.initialize({
      API_URL: baseUrl,
      AGENT_ID: "agent"
    });
  });
</script>
```

## gRPC Agent 能力

`proto/agent.proto` 当前包含：

- `RegisterAgent`
- `Heartbeat`
- `ListAgents`
- `AssignSession`
- `ReleaseSession`

当前实现仍是单体服务内嵌 gRPC server，不是独立 agent 平台。

## 自动回复规则

- 支持多个关键词，分隔符支持 `,`、`，`、`;`、`；`、`|` 和换行
- 关键词会做去重、去空白、转小写归一化
- 先做精确匹配
- 再做最长关键词优先的包含匹配
- 缓存优先走 Redis
- Redis 不可用时退化到内存缓存

## 测试

后端快速检查：

```bash
go test ./...
```

agent 调度对比基准：

```bash
go test ./agent -bench DispatcherAssignSession -benchmem
```

前端构建检查：

```bash
npm run build
```
