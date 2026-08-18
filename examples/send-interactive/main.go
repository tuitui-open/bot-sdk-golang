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
	response, err := client.IM.SendInteractive(context.Background(), tuitui.SendIMInteractiveOptions{
		To: target,
		Interactive: tuitui.InteractiveMessage{
			"head":   map[string]interface{}{"text": "Go SDK 卡片"},
			"body":   map[string]interface{}{"content": "请选择"},
			"action": []interface{}{map[string]interface{}{"text": "确认", "name": "confirm", "value": "confirm"}},
		},
	})
	if err != nil {
		log.Printf("交互消息发送失败: %v", err)
	} else {
		sample.PrintResponse("交互消息发送成功", response)
	}
}
