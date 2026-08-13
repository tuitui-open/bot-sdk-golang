package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPropertyTeamsAndFileSpaceFlows(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		responses := map[string]interface{}{
			"/prop/get":                   map[string]interface{}{"errcode": 0, "robot_name": " Bot ", "robot_uid": 123, "robot_account": " bot@example "},
			"/shortcutCommand/get":        map[string]interface{}{"errcode": 0, "datas": map[string]interface{}{"shortcut_cmds": []interface{}{map[string]interface{}{"command_name": "help", "command_content": "/help", "command_description": "Help"}}}},
			"/teams/channel/info":         map[string]interface{}{"errcode": 0, "datas": map[string]interface{}{"info": map[string]interface{}{"team_id": "team", "name": "channel", "announcement": "notice"}}},
			"/teams/channel/postTag/list": map[string]interface{}{"errcode": 0, "datas": map[string]interface{}{"tags": []interface{}{map[string]interface{}{"tag_id": "tag-1", "name": "News"}}}},
			"/teams/member/list":          map[string]interface{}{"errcode": 0, "datas": map[string]interface{}{"members": []interface{}{map[string]interface{}{"name": "Alice"}, map[string]interface{}{"name": "Bob"}}}},
			"/file_space/node/list":       map[string]interface{}{"errcode": 0, "datas": map[string]interface{}{"list": []interface{}{map[string]interface{}{"node_id": "file", "node_type": NodeTypeFile, "name": "a.txt"}}}},
			"/message/custom/send":        map[string]interface{}{"errcode": 0, "post_id": "post"},
		}
		value, exists := responses[request.URL.Path]
		if !exists {
			value = map[string]interface{}{"errcode": 0}
		}
		_ = json.NewEncoder(writer).Encode(value)
	}))
	defer server.Close()
	client, _ := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	ctx := context.Background()
	info, err := client.Property.Info(ctx)
	if err != nil || info.Name != "Bot" || info.UID != "123" || info.Account != "bot@example" {
		t.Fatalf("unexpected bot info: %#v, %v", info, err)
	}
	commands, err := client.Property.GetShortcutCommands(ctx)
	if err != nil || len(commands) != 1 || commands[0].Name != "help" {
		t.Fatalf("unexpected commands: %#v, %v", commands, err)
	}
	channel, err := client.Teams.GetChannelInfo(ctx, "channel")
	if err != nil || channel["team_id"] != "team" {
		t.Fatalf("unexpected channel: %#v, %v", channel, err)
	}
	if announcement, _ := client.Teams.GetAnnouncement(ctx, "channel"); announcement != "notice" {
		t.Fatalf("unexpected announcement: %#v", announcement)
	}
	tags, _ := client.Teams.GetChannelPostTags(ctx, "channel")
	if len(tags) != 1 || tags[0] != "News" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
	if _, err := client.Teams.SendPost(ctx, SendPostOptions{TeamID: "team", ChannelID: "channel", Text: "hello", Tag: "news"}); err != nil {
		t.Fatal(err)
	}
	names, _ := client.Teams.GetMemberNames(ctx, "team")
	if names != "Alice\nBob" {
		t.Fatalf("unexpected names: %q", names)
	}
	files, err := client.FileSpace.ListFiles(ctx, FileSpaceContext{SpaceID: "team"})
	if err != nil || len(files) != 1 || files[0].Filename != "a.txt" {
		t.Fatalf("unexpected files: %#v, %v", files, err)
	}
}

func TestUploadAndInteractiveResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/media/upload" {
			if request.URL.Query().Get("type") != "image" {
				t.Errorf("unexpected upload type: %s", request.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 0, "media_id": "media"})
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 0, "msgids": []interface{}{map[string]interface{}{"msgid": "message"}}})
	}))
	defer server.Close()
	client, _ := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	ctx := context.Background()
	uploaded, err := client.File.Upload(ctx, []byte("png"), &UploadOptions{Filename: "a.png", ContentType: "image/png"})
	if err != nil || uploaded.FID != "media" || uploaded.MediaType != "image" {
		t.Fatalf("unexpected upload: %#v, %v", uploaded, err)
	}
	response, err := client.IM.SendInteractive(ctx, SendIMInteractiveOptions{To: client.To.Account("alice"), Interactive: InteractiveMessage{"body": map[string]interface{}{"content": "hello"}}})
	if err != nil || response["msgid"] != "message" {
		t.Fatalf("unexpected response: %#v, %v", response, err)
	}
}

func TestEventRenderingAndMediaExtraction(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"msg_type": "mixed", "text": "hello", "images": []interface{}{"https://example/a.png"}, "files": []interface{}{map[string]interface{}{"url": "https://example/a.pdf"}}}
	if rendered := RenderMessageBody(data); rendered != "hello\n[图片] https://example/a.png" {
		t.Fatalf("unexpected rendered body: %q", rendered)
	}
	media := GetMessageMedia(data)
	if len(media) != 2 || media[0].URL != "https://example/a.png" || media[1].URL != "https://example/a.pdf" {
		t.Fatalf("unexpected media: %#v", media)
	}
}
