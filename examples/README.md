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

## 上传并查询文件

上传示例文件，并输出文件 ID 和临时下载 URL：

```
go run ./examples/file-op
```

## 接收事件

```
go run ./examples/receive
```

连接成功后持续输出事件，按 `Ctrl+C` 退出。

## Agent 执行中间步骤上报

```
go run ./examples/agent-report

# 子 Agent 上报示例
go run ./examples/agent-report --subagent
```

## 搜索用户

```
go run ./examples/user-search 张三
```

仅需机器人凭证，输出命中的用户数组，不建立事件订阅。

## 直接发送文件 URL

以下示例使用已创建的客户端；请替换为可下载的真实文件 URL。

```go
response, err := client.IM.SendFileByURL(ctx, tuitui.SendIMFileByURLOptions{
    To: client.To.Account("alice"),
    URL: "https://example.com/report.pdf",
    Filename: "季度报告.pdf",
})
```

文件名可省略，默认由服务端取 URL 路径中的文件名。此入口直接发送 IM 文件消息，SDK 不下载、上传或自动识别图片；内网 URL 受服务端 SSRF 策略限制。原有文件发送方法保留下载、识别并上传来源的行为。
