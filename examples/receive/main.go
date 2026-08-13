package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

func main() {
	if err := dotenv.LoadClosest(); err != nil {
		log.Fatal(err)
	}
	appID := os.Getenv("TUITUI_BOT_APPID")
	appSecret := os.Getenv("TUITUI_BOT_SECRET")
	client, err := tuitui.NewClient(appID, appSecret, nil)
	if err != nil {
		log.Fatal(err)
	}
	client.Event.Subscribe(context.Background(), &tuitui.SubscribeOptions{
		OnConnected: func() {
			log.Printf("tuitui bot(%s) 已连接websocket，正在监听事件", appID)
		},
		OnEvent: func(event tuitui.RawEvent) {
			body := event["body"].(map[string]interface{})
			eventName := body["event"].(string)
			rawJSON, _ := json.MarshalIndent(event, "", "  ")
			log.Printf("收到事件:%s\n%s", eventName, rawJSON)
			printMessage(eventName, body, client.Event)
		},
		OnError: func(err error) {
			log.Printf("tuitui websocket: %v", err)
		},
	})
	select {}
}

func printMessage(eventName string, body map[string]interface{}, eventAPI *tuitui.EventAPI) {
	switch eventName {
	case tuitui.EventSingleChat:
		data := body["data"].(map[string]interface{})
		log.Printf(
			"收到单聊消息，来自:%s 解析文本:%s",
			body["user_account"],
			eventAPI.RenderMessageBody(data),
		)
	case tuitui.EventGroupChat:
		data := body["data"].(map[string]interface{})
		log.Printf(
			"收到群聊(%s)消息，来自:%s 是否@自己:%t 解析文本:%s",
			body["group_id"],
			body["user_account"],
			data["at_me"],
			eventAPI.RenderMessageBody(data),
		)
	case tuitui.EventTeamsPostCreate:
		data := body["data"].(map[string]interface{})
		log.Printf(
			"收到团队(%s)-频道(%s)消息，来自:%s 是否@自己:%t 解析文本:%s",
			data["team_id"],
			data["channel_id"],
			body["user_account"],
			data["at_me"],
			data["content"],
		)
	}
}
