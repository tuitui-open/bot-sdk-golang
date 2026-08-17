//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

const imageDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

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
	response, err := client.Teams.SendPost(ctx, tuitui.SendPostOptions{TeamID: os.Getenv("TARGET_TEAM"), ChannelID: os.Getenv("TARGET_CHANNEL"), Text: "**Go SDK** " + runID})
	if err != nil {
		t.Fatal(err)
	}
	postID := requireResponseID(t, response, "post_id", "msgid", "message_id")
	if _, err := client.Teams.SendPost(ctx, tuitui.SendPostOptions{
		TeamID:    os.Getenv("TARGET_TEAM"),
		ChannelID: os.Getenv("TARGET_CHANNEL"),
		Text:      "**Go SDK 引用回帖** " + runID,
		ParentID:  postID,
		RefPostID: postID,
	}); err != nil {
		t.Fatal(err)
	}
	space := tuitui.FileSpaceContext{SpaceID: os.Getenv("TARGET_TEAM"), SpaceType: tuitui.SpaceTypeTeam}
	if _, err := client.FileSpace.ListNodes(ctx, space); err != nil {
		t.Fatal(err)
	}
}

func TestTeams发送频道图片和文件(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_TEAM", "TARGET_CHANNEL")
	client, err := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	teamID := os.Getenv("TARGET_TEAM")
	channelID := os.Getenv("TARGET_CHANNEL")
	runID := fmt.Sprintf("go-teams-file-%d", time.Now().UnixNano())

	mediaText := "[Go SDK E2E channel media] " + runID
	response, err := client.Teams.SendFile(ctx, tuitui.SendPostFileOptions{
		TeamID:    teamID,
		ChannelID: channelID,
		Source:    imageDataURL,
		Text:      mediaText,
	})
	if err != nil {
		t.Fatal(err)
	}
	mediaPostID := requireResponseID(t, response, "post_id", "msgid", "message_id")
	mediaChain, err := client.Teams.GetPostChainForAgent(ctx, teamID, channelID, mediaPostID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mediaChain) == 0 || !strings.Contains(mediaChain[0].Content, mediaText) {
		t.Fatalf("频道图片帖子缺少测试文本：%#v", mediaChain)
	}

	fileText := "[Go SDK E2E channel file] " + runID
	filename := "go-sdk-e2e-" + runID + ".md"
	fileContent := []byte("# Go SDK E2E Markdown\n\n这是由自动化测试生成的 Markdown 文件。\n\nRun ID: " + runID + "\n")
	response, err = client.Teams.SendFile(ctx, tuitui.SendPostFileOptions{
		TeamID:      teamID,
		ChannelID:   channelID,
		Source:      fileContent,
		Filename:    filename,
		ContentType: "text/markdown",
		Text:        fileText,
	})
	if err != nil {
		t.Fatal(err)
	}
	filePostID := requireResponseID(t, response, "post_id", "msgid", "message_id")
	fileChain, err := client.Teams.GetPostChainForAgent(ctx, teamID, channelID, filePostID)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileChain) == 0 || !strings.Contains(fileChain[0].Content, fileText) {
		t.Fatalf("频道文件帖子缺少测试文本：%#v", fileChain)
	}
	properties, err := json.Marshal(fileChain[0].Properties)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(properties), filename) {
		t.Fatalf("频道文件属性缺少文件名 %q：%s", filename, properties)
	}
}

func requireResponseID(t *testing.T, response tuitui.APIResponse, keys ...string) string {
	t.Helper()
	containers := []map[string]interface{}{response}
	for _, containerKey := range []string{"datas", "data", "result"} {
		if container, ok := response[containerKey].(map[string]interface{}); ok {
			containers = append(containers, container)
		}
	}
	for _, container := range containers {
		for _, key := range keys {
			if id := responseIDValue(container[key]); id != "" {
				return id
			}
		}
		for _, arrayKey := range []string{"msgids", "post_ids"} {
			values, _ := container[arrayKey].([]interface{})
			if len(values) == 0 {
				continue
			}
			if id := responseIDValue(values[0]); id != "" {
				return id
			}
			if item, ok := values[0].(map[string]interface{}); ok {
				for _, key := range keys {
					if id := responseIDValue(item[key]); id != "" {
						return id
					}
				}
			}
		}
	}
	t.Fatalf("响应中缺少 ID 字段：%v", keys)
	return ""
}

func responseIDValue(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case float64:
		return fmt.Sprint(value)
	default:
		return ""
	}
}
