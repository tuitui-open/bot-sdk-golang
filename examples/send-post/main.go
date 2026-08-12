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
	teamID, channelID, err := sample.ParsePostTarget(os.Args[1:])
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
	response, err := client.Teams.SendPost(context.Background(), tuitui.SendPostOptions{
		TeamID: teamID, ChannelID: channelID, Text: "**来自 Go SDK 的帖子**",
	})
	if err != nil {
		log.Fatal(err)
	}
	sample.PrintResponse("频道帖子发送成功", response)
}
