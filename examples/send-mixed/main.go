package main

import (
	"context"
	"log"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/examples/internal/sample"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

func main() {
	target, err := sample.ParseTarget(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	if err := dotenv.LoadClosest(); err != nil {
		log.Fatal(err)
	}
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	response, err := client.IM.SendMixed(context.Background(), tuitui.SendIMMixedOptions{
		To: target,
		Items: []tuitui.MixedItem{
			{Type: "text", Text: "图片如下："},
			{Type: "image", Source: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="},
			{Type: "text", Text: "来自 Go SDK 的 Mixed 消息"},
		},
	})
	if err != nil {
		log.Printf("Mixed 消息发送失败: %v", err)
	} else {
		sample.PrintResponse("Mixed 消息发送成功", response)
	}
}
