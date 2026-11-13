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

func TestUser按关键词搜索并返回用户数组(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET")
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	users, err := client.User.Search(ctx, "zhq")
	if err != nil {
		t.Fatal(err)
	}
	if users == nil {
		t.Fatal("应返回用户数组")
	}
	t.Logf("zhq 命中 %d 个用户", len(users))
}
