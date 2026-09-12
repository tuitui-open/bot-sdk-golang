package main

import (
	"context"
	"log"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/examples/internal/sample"
)

func main() {
	teamID, channelID, err := sample.ParsePostTarget(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	appID, appSecret, options, configErr := sample.LoadConfig()
	if configErr != nil {
		log.Fatal(configErr)
	}
	client := tuitui.NewClient(appID, appSecret, options)
	response, err := client.Teams.SendPost(context.Background(), tuitui.SendPostOptions{
		TeamID: teamID, ChannelID: channelID, Text: "**来自 Go SDK 的帖子**",
	})
	if err != nil {
		log.Printf("频道帖子发送失败: %v", err)
	} else {
		sample.PrintResponse("频道帖子发送成功", response)
	}
}
