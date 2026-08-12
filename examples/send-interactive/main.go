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
	client, err := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.IM.SendInteractive(context.Background(), tuitui.SendIMInteractiveOptions{
		To: target,
		Interactive: tuitui.InteractiveMessage{
			"head":   map[string]any{"text": "Go SDK 卡片"},
			"body":   map[string]any{"content": "请选择"},
			"action": []any{map[string]any{"text": "确认", "name": "confirm", "value": "confirm"}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	sample.PrintResponse("交互消息发送成功", response)
}
