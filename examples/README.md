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

连接成功后持续输出原始事件，按 `Ctrl+C` 取消订阅并退出。
