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
	client := tuitui.NewClient(appID, appSecret, options)
	response, err := client.IM.SendFile(context.Background(), tuitui.SendIMFileOptions{
		To: target, Source: "examples/README.md",
	})
	if err != nil {
		log.Printf("文件发送失败: %v", err)
	} else {
		sample.PrintResponse("文件发送成功", response)
	}
}
