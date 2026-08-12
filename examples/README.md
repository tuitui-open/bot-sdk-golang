# SDK 使用示例

本目录提供接收事件，以及发送 IM 文本、文件、Mixed、交互消息和频道帖子的可运行示例。

## 运行准备

Monorepo 开发者直接使用仓库根 `.env`。发布包用户在 `examples` 目录将 `.env.example` 复制为
`.env` 并从该目录运行；已有进程环境变量优先于 `.env` 中的同名值。

Sample 配置只包含机器人凭证，发送目标通过命令行参数指定：

```dotenv
TUITUI_BOT_APPID=机器人 App ID
TUITUI_BOT_SECRET=机器人 App Secret
```

以下命令均从 Go 项目目录运行。

## 发送 IM 消息

账号、UID 和群 ID 三种目标必须且只能指定一种：

```
go run ./examples/send-text --account alice
go run ./examples/send-text --uid 123456
go run ./examples/send-text --group 987654

go run ./examples/send-file --account alice
go run ./examples/send-mixed --account alice
go run ./examples/send-interactive --account alice
```

`send-file` 发送 `examples/README.md`，`send-mixed` 使用内嵌的小型 PNG，无需额外准备文件。

## 发送频道帖子

```
go run ./examples/send-post --team 123456 --channel 789012
```

## 接收事件

```
go run ./examples/receive
```

连接成功后持续输出原始事件，按 `Ctrl+C` 取消订阅并退出。

所有发送示例都会在成功后输出服务端返回的完整 JSON。参数错误和请求失败会以非零退出码结束。
这些示例会调用真实推推环境，真实 `.env` 已被 Git 忽略，不得提交。
