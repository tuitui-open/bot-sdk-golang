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
	client := tuitui.NewClient(
		os.Getenv("TUITUI_BOT_APPID"),
		os.Getenv("TUITUI_BOT_SECRET"),
		nil,
	)
	response, err := client.IM.SendText(context.Background(), tuitui.SendIMTextOptions{
		To:   target,
		Text: "你好，来自 `go` SDK",
	})
	if err != nil {
		log.Printf("消息发送失败: %v", err)
	} else {
		sample.PrintResponse("消息发送成功", response)
	}
}
