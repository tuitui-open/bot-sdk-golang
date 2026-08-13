package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

func main() {
	if err := dotenv.LoadClosest(); err != nil {
		log.Fatal(err)
	}
	client, err := tuitui.NewClient(
		os.Getenv("TUITUI_BOT_APPID"),
		os.Getenv("TUITUI_BOT_SECRET"),
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	go func() {
		<-interrupt
		cancel()
	}()
	subscription := client.Event.Subscribe(ctx, &tuitui.SubscribeOptions{
		OnConnected: func() { log.Println("WebSocket 已连接") },
		OnEvent:     func(event tuitui.RawEvent) { fmt.Printf("%#v\n", event) },
		OnError:     func(err error) { log.Printf("WebSocket 错误: %v", err) },
	})
	<-ctx.Done()
	subscription.Unsubscribe()
}
