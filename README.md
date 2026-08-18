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
- `client.Property`：机器人自身属性查询与修改（名称、账号、头像、Webhook、可交互式消息回调地址和快捷指令）。
- `client.Event`：通过 WebSocket 订阅推推事件，用于实时收消息等场景。
- `client.Request`：调用尚未封装的原始 Bot API。

消息、帖子的内容支持 Markdown 格式（可交互式消息除外）。

## 示例

包含收、发消息示例，用法详见 [`examples/README.md`](examples/README.md)。
