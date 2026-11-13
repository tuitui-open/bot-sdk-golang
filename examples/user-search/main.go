package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		log.Fatal("用法: go run ./examples/user-search <搜索关键词>")
	}
	keyword := os.Args[1]
	if err := dotenv.LoadClosest(); err != nil {
		log.Fatal(err)
	}
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	users, err := client.User.Search(context.Background(), keyword)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
