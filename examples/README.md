# SDK 使用示例

- 发送：文本、文件（含图片）、图文混发、可交互消息、频道帖子
- 接收消息（事件）

## 运行准备

配置环境变量 `TUITUI_BOT_APPID` 和 `TUITUI_BOT_SECRET`，也支持 `.env` 方式，参考 [`.env.example`](.env.example) 文件。

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

## 发送频道帖子

```
go run ./examples/send-post --team 123456 --channel 789012
```

## 接收事件

```
go run ./examples/receive
```

连接成功后持续输出事件，按 `Ctrl+C` 退出。

## Agent 执行过程

```
go run ./examples/agent-report
```

示例先查询机器人信息，再监听私聊、@Bot 群聊和 @Bot 团队帖子，未 @ 的群消息和帖子会忽略。
每条消息独立执行 12 条主/子事件，主模型和工具分别模拟 2 秒、1 秒；其他消息可同时进入流程。
Ctrl+C 取消订阅并清理示例任务，后台结果通过 logger 输出，退出不承诺排空上报队列。

所有示例均支持可选 `TUITUI_BOT_API_BASE_URL`（http/https）和
`TUITUI_BOT_WEBSOCKET_BASE_URL`（ws/wss）完整 URL；不设置时使用 SDK 默认地址。
最近 `.env` 仅补齐未设置的进程环境变量。
