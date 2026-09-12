# 推推机器人 Go SDK

推推机器人 Go SDK，移植自 TypeScript 包 `@qihoo/tuitui-bot-sdk`。

## 安装

```
go get github.com/tuitui-open/bot-sdk-golang
```

最低 Go 版本为 1.12。

## 发送消息

```go
package main

import (
    "context"
    "log"

    tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func main() {
    client := tuitui.NewClient("your-appid", "your-secret", nil)
    response, err := client.IM.SendText(context.Background(), tuitui.SendIMTextOptions{
        To:   client.To.Account("接收账号"),
        Text: "你好，来自 `go` SDK",
    })
    if err != nil {
        log.Printf("消息发送失败: %v", err)
    } else {
        log.Printf("消息发送成功: %#v", response)
    }
}
```

所有网络方法都接收 `context.Context`，调用方可按需取消请求；不需要主动取消时传入
`context.Background()` 即可。HTTP 请求默认超时时间为 30 秒，也可通过 `ClientOptions.HTTPTimeout` 调整。

## API

- `client.IM`：发送单聊、群聊消息（文本、图片、图文、页面、链接、文件和交互卡片）、编辑消息、表情回复和拉取聊天记录。
- `client.Teams`：团队、频道、帖子 API。
- `client.File`：底层公共文件 API，可用于消息、帖子等场景。
- `client.FileSpace`：文件空间，目前仅用于团队模块，包含文件、目录的新增、列表和删除。
- `client.Group`：建群、群成员管理及群信息查询。
- `client.Property`：机器人自身属性查询与修改（名称、账号、头像、Webhook、可交互式消息回调地址和快捷指令）。
- `client.Event`：通过 WebSocket 订阅推推事件，用于实时收消息等场景。
- `client.Agent`：构造主/子 Agent 上下文并在后台按序上报执行过程。
- `client.Request`：调用尚未封装的原始 Bot API。

消息、帖子的内容支持 Markdown 格式（可交互式消息除外）。

## 示例

包含收、发消息示例，用法详见 [`examples/README.md`](examples/README.md)。

## Agent 过程上报

```go
ctx, err := client.Agent.BuildContext(tuitui.BuildAgentContextOptions{
    Target: tuitui.AgentContextTarget{Type: "group", GroupID: "群 ID"},
    MessageID: "消息 ID",
})
if err != nil { panic(err) }
client.Agent.Report(context.Background(), tuitui.AgentMessageReceivedEvent{
    Context: ctx,
    Data: tuitui.AgentEventMessageReceivedData{Content: "开始处理"},
})
```

`ContextFromMessage(body, nil)` 从单聊、群聊、团队主帖或回帖构造上下文；
`BuildSubagentContext(BuildAgentSubagentContextOptions{RequesterContext: ctx})` 自动生成子 ID。
`AgentEventSubagentContext` 可直接用于 LLM 事件的 `Context`，工具事件使用
`AgentEventToolContext{Context: sub, ToolCallID: "调用 ID"}`，完整保留父关联字段。
Agent 目标的 `direct/group/channel` 和 `sessionKey` 是上报协议字段，不用于 IM 目标转换。

八种事件结构体为 `AgentMessageReceivedEvent`、`AgentLLMInputEvent`、`AgentLLMOutputEvent`、
`AgentBeforeToolCallEvent`、`AgentAfterToolCallEvent`、`AgentSubagentSpawnedEvent`、
`AgentSubagentEndedEvent` 和 `AgentEndEvent`，分别对应 `message_received`、`llm_input`、
`llm_output`、`before_tool_call`、`after_tool_call`、`subagent_spawned`、`subagent_ended`、`agent_end`。
每种事件的数据使用对应 `AgentEvent...Data`，可通过 `Extensions` 保留附加字段；附加字段不得覆盖声明字段。
`Thinking`、`Result` 的 nil 表示 JSON null，false、0 和非 nil 空切片保留原值。

`Report` 同步校验并入队，无返回值；非法参数仅记失败日志。每个 Client 独立串行发送，失败不重试，
下一条继续；每条请求使用自己的 context，取消一条不取消其他事件。可选的 `ClientOptions.Logger`
收到包含完整响应的成功日志和错误详情；日志 panic 不改变结果。未设置 logger 时静默。
调用方需保持进程和 context 存活；SDK 不提供 flush、投递保证或退出时自动排空。
指定大字段超过 500 个 Unicode code point 时裁剪，prompt、thinking 和普通扩展字段不裁剪。

示例 `go run ./examples/agent-report` 持续接收消息，完整演示 12 条主/子事件。
默认 `go test ./...` 仅执行 Mock，不加载真实配置。Agent E2E 显式运行：

```
go test -tags e2e ./tests/e2e -run TestAgent -v
```

E2E 从仓库根 `.env` 读取 `TUITUI_BOT_APPID`、`TUITUI_BOT_SECRET`、`TARGET_GROUP` 和可选
`TUITUI_BOT_API_BASE_URL`，发送一条真实群消息并检查全部 12 条成功响应，无需第二 Bot 或订阅。
