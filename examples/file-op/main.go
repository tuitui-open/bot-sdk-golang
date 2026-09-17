package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

func main() {
	if err := dotenv.LoadClosest(); err != nil {
		log.Fatal(err)
	}
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	uploaded, err := client.File.Upload(context.Background(), "examples/README.md", nil)
	if err != nil {
		log.Printf("文件操作失败: %v", err)
		return
	}
	temporaryURL, err := client.File.Query(context.Background(), uploaded.FID)
	if err != nil {
		log.Printf("文件操作失败: %v", err)
		return
	}
	fmt.Println("文件上传成功")
	fmt.Printf("fid: %s\n", uploaded.FID)
	fmt.Printf("temporaryUrl: %s\n", temporaryURL)
}
