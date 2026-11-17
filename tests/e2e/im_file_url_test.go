//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func TestIM直接发送文件URL和可选文件名(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_ACCOUNT", "TARGET_GROUP")
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	uploaded, err := client.File.Upload(ctx, []byte("URL 文件发送"), &tuitui.UploadOptions{Filename: "url.txt"})
	if err != nil {
		t.Fatal(err)
	}
	url, err := client.File.Query(ctx, uploaded.FID)
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range []tuitui.SendIMFileByURLOptions{
		{To: client.To.Account(os.Getenv("TARGET_ACCOUNT")), URL: url},
		{To: client.To.Group(os.Getenv("TARGET_GROUP")), URL: url, Filename: "自定义文件名.txt"},
	} {
		if _, err := client.IM.SendFileByURL(ctx, options); err != nil {
			t.Fatal(err)
		}
	}
}
