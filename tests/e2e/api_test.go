//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func TestPropertyAndIM(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_ACCOUNT", "TARGET_GROUP")
	client, err := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := client.Property.Info(ctx); err != nil {
		t.Fatal(err)
	}
	runID := fmt.Sprintf("go-im-%d", time.Now().UnixNano())
	if _, err := client.IM.SendText(ctx, tuitui.SendIMTextOptions{To: client.To.Account(os.Getenv("TARGET_ACCOUNT")), Text: "**Go SDK** `" + runID + "`"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.IM.SendText(ctx, tuitui.SendIMTextOptions{To: client.To.Group(os.Getenv("TARGET_GROUP")), Text: "**Go SDK group** `" + runID + "`"}); err != nil {
		t.Fatal(err)
	}
}

func TestTeamsAndFileSpace(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_TEAM", "TARGET_CHANNEL")
	client, err := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runID := fmt.Sprintf("go-teams-%d", time.Now().UnixNano())
	if _, err := client.Teams.SendPost(ctx, tuitui.SendPostOptions{TeamID: os.Getenv("TARGET_TEAM"), ChannelID: os.Getenv("TARGET_CHANNEL"), Text: "**Go SDK** " + runID}); err != nil {
		t.Fatal(err)
	}
	space := tuitui.FileSpaceContext{SpaceID: os.Getenv("TARGET_TEAM"), SpaceType: tuitui.SpaceTypeTeam}
	if _, err := client.FileSpace.ListNodes(ctx, space); err != nil {
		t.Fatal(err)
	}
}
