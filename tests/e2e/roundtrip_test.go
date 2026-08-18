//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func TestTwoBotRoundtrip(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TUITUI_BOT2_APPID", "TUITUI_BOT2_SECRET", "TARGET_GROUP")
	bot1 := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	bot2 := tuitui.NewClient(os.Getenv("TUITUI_BOT2_APPID"), os.Getenv("TUITUI_BOT2_SECRET"), nil)
	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeoutSeconds*time.Second)
	defer cancel()
	identity, err := bot1.Property.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	connected := make(chan struct{}, 1)
	received := make(chan string, 1)
	token := fmt.Sprintf("go-roundtrip-%d", time.Now().UnixNano())
	subscription := bot1.Event.Subscribe(ctx, &tuitui.SubscribeOptions{
		OnConnected: func() { connected <- struct{}{} },
		OnEvent: func(body tuitui.EventBody) {
			if fmt.Sprint(body["event"]) == tuitui.EventGroupChat {
				if data, ok := body["data"].(map[string]interface{}); ok && fmt.Sprint(data["text"]) == token {
					select {
					case received <- body.BotName():
					default:
					}
				}
			}
		},
	})
	defer subscription.Unsubscribe()
	select {
	case <-connected:
	case <-ctx.Done():
		t.Fatal("WebSocket connection timeout")
	}
	if _, err := bot2.IM.SendText(ctx, tuitui.SendIMTextOptions{To: bot2.To.Group(os.Getenv("TARGET_GROUP")), Text: token}); err != nil {
		t.Fatal(err)
	}
	select {
	case botName := <-received:
		if botName != identity.Name {
			t.Fatalf("消息事件机器人名称不正确：%q", botName)
		}
	case <-ctx.Done():
		t.Fatal("message event timeout")
	}
}
