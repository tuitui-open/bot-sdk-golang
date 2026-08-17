package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test共享帖子按时间顺序输出正文和媒体(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/teams/share/post" {
			t.Fatalf("请求路径错误：%s", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{
			"errcode": 0,
			"datas": map[string]interface{}{
				"topic": map[string]interface{}{
					"from_name":   "作者",
					"create_time": 1723420800000,
					"content":     "主贴\n\n[图片] \n[文件] ",
					"properties": map[string]interface{}{
						"images": []interface{}{map[string]interface{}{"name": "主图.png", "url": "https://example.com/topic.png"}},
						"files":  []interface{}{map[string]interface{}{"name": "主文件.txt", "url": "https://example.com/topic.txt", "fid": "topic-file"}},
					},
				},
				"reply_list": []interface{}{
					map[string]interface{}{
						"from_name": "新回复",
						"content":   "第二条回复",
						"properties": map[string]interface{}{
							"files": []interface{}{map[string]interface{}{"name": "新文件.txt", "url": "https://example.com/new.txt", "file_id": "new-file"}},
						},
					},
					map[string]interface{}{
						"from_name": "旧回复",
						"content":   "第一条回复",
						"properties": map[string]interface{}{
							"images": []interface{}{map[string]interface{}{"name": "旧图.png", "url": "https://example.com/old.png"}},
						},
					},
				},
			},
		})
	}))
	defer server.Close()
	client, err := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	text, err := client.Teams.GetSharedPostForAgent(context.Background(), "share")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"[图片] 主图.png: https://example.com/topic.png",
		"[文件] 主文件.txt: https://example.com/topic.txt",
		"[图片] 旧图.png: https://example.com/old.png",
		"[文件] 新文件.txt: https://example.com/new.txt",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("共享帖子缺少 %q：\n%s", expected, text)
		}
	}
	if strings.Index(text, "主贴") > strings.Index(text, "第一条回复") || strings.Index(text, "第一条回复") > strings.Index(text, "第二条回复") {
		t.Fatalf("共享帖子顺序错误：\n%s", text)
	}
	if strings.Count(text, "[图片]") != 2 || strings.Count(text, "[文件]") != 2 {
		t.Fatalf("共享帖子仍包含空媒体占位符：\n%s", text)
	}

	message := map[string]interface{}{"msg_type": "shared_post", "shared_post": "share"}
	subscription := Subscription{ctx: context.Background(), teams: client.Teams}
	if err := subscription.normalizeSharedPost(message); err != nil {
		t.Fatal(err)
	}
	media := GetDetailedMessageMedia(message)
	if len(media) != 4 {
		t.Fatalf("共享帖子媒体数量错误：%#v", media)
	}
	if media[0].Name != "主图.png" || media[1].FileID != "topic-file" || media[2].Name != "旧图.png" || media[3].FileID != "new-file" {
		t.Fatalf("共享帖子媒体顺序或元数据错误：%#v", media)
	}
}

func TestSendPost发送引用回帖字段(t *testing.T) {
	t.Parallel()
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/message/custom/send" {
			t.Fatalf("请求路径错误：%s", request.URL.Path)
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 0})
	}))
	defer server.Close()
	client, err := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Teams.SendPost(context.Background(), SendPostOptions{
		TeamID:    "team",
		ChannelID: "channel",
		Text:      "reply",
		ParentID:  "parent",
		RefPostID: "quoted-reply",
	})
	if err != nil {
		t.Fatal(err)
	}
	target := payload["toteams"].([]interface{})[0].(map[string]interface{})
	if target["parent_id"] != "parent" || target["ref_post_id"] != "quoted-reply" {
		t.Fatalf("团队回帖目标错误：%#v", target)
	}
}
