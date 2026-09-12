package main

import (
	"context"
	"log"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/examples/internal/sample"
)

func main() {
	target, err := sample.ParseTarget(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	appID, appSecret, options, configErr := sample.LoadConfig()
	if configErr != nil {
		log.Fatal(configErr)
	}
	client := tuitui.NewClient(
		appID,
		appSecret,
		options,
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
