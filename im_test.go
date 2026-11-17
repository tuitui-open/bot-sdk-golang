package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestIM直接发送文件URL和可选文件名(t *testing.T) {
	var bodies []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/custom/send" {
			t.Errorf("不应下载或上传文件: %s", r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	url := "https://example.com/download?key=1"
	for _, options := range []SendIMFileByURLOptions{
		{To: client.To.Account("alice"), URL: url},
		{To: client.To.Group("g1"), URL: url, Filename: "季度报告.pdf", At: []string{"bob"}},
	} {
		if _, err := client.IM.SendFileByURL(context.Background(), options); err != nil {
			t.Fatal(err)
		}
	}
	if len(bodies) != 2 {
		t.Fatalf("请求数量错误: %d", len(bodies))
	}
	if !reflect.DeepEqual(bodies[0]["attachment"], map[string]interface{}{"media_id": url}) ||
		!reflect.DeepEqual(bodies[1]["attachment"], map[string]interface{}{"media_id": url, "filename": "季度报告.pdf"}) ||
		bodies[0]["msgtype"] != "attachment" || bodies[1]["msgtype"] != "attachment" ||
		!reflect.DeepEqual(bodies[1]["at"], []interface{}{"bob"}) {
		t.Fatalf("文件消息协议错误: %#v", bodies)
	}
	if _, err := client.IM.SendFileByURL(context.Background(), SendIMFileByURLOptions{
		To: client.To.Account("alice"), URL: url, At: []string{"bob"},
	}); err == nil {
		t.Fatal("单聊不应接受群聊提及")
	}
}
