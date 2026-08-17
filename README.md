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
    "os"

    tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func main() {
    client, err := tuitui.NewClient(
        os.Getenv("TUITUI_BOT_APPID"),
        os.Getenv("TUITUI_BOT_SECRET"),
        nil,
    )
    if err != nil {
        log.Fatal(err)
    }
    _, err = client.IM.SendText(context.Background(), tuitui.SendIMTextOptions{
        To:   client.To.Account("接收账号"),
        Text: "**Hello from Go**",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

所有网络方法都接收 `context.Context`。服务端非零 `errcode`、非 2xx 响应、无效 JSON 和网络错误
统一返回 `*tuitui.APIError`。

## 主要领域

- `client.IM`：文本、图文混排、页面、链接、文件、交互卡片、编辑、表情回复和聊天记录。
- `client.Teams`：帖子、回复、编辑、标签、媒体、成员、频道、帖子链和历史。
- `client.File`：本地路径、字节、Reader、Data URL 和远程 URL 上传。
- `client.FileSpace`：节点、扁平文件、多级目录上传和团队文件。
- `client.Property`：机器人身份、名称、头像、Webhook、交互地址和快捷指令。
- `client.Event`：WebSocket 订阅、ACK、去重、心跳、重连、事件正文和媒体提取。
- `client.Request`：调用尚未封装的原始 Bot API。

## 示例

设置环境变量 `TUITUI_BOT_APPID` 和 `TUITUI_BOT_SECRET` 后，可直接运行 `examples` 中的示例。完整说明见 [`examples/README.md`](examples/README.md)。
