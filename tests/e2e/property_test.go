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

func TestProperty设置并读取带Tag的快捷指令(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET")
	client, err := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	readableTime := now.Format("2006-01-02 15:04:05")
	command := tuitui.ShortcutCommand{
		Name:        fmt.Sprintf("golang-sdk-%s", now.Format("20060102-150405")),
		Content:     fmt.Sprintf("/golang-sdk %s", readableTime),
		Description: fmt.Sprintf("Go SDK E2E %s", readableTime),
		Tag:         "golang",
	}
	noAt := false
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := client.Property.SetShortcutCommands(
		ctx,
		[]tuitui.ShortcutCommand{command},
		&tuitui.SetShortcutCommandsOptions{NoAt: &noAt},
	); err != nil {
		t.Fatal(err)
	}

	commands, err := client.Property.GetShortcutCommands(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, actual := range commands {
		if actual == command {
			return
		}
	}
	t.Fatalf("未读取到本次设置的快捷指令：want=%#v, got=%#v", command, commands)
}
