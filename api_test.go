package tuitui

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewClient允许空凭证(t *testing.T) {
	client := NewClient("", "", nil)
	if client.config.appID != "" || client.config.appSecret != "" {
		t.Fatalf("客户端未保留空凭证：%#v", client.config)
	}
}

func TestSendStrongNoticeAndPhoneAlarm(t *testing.T) {
	t.Parallel()
	var paths []string
	var payloads []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		var payload map[string]interface{}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		payloads = append(payloads, payload)
		_, _ = writer.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	ctx := context.Background()

	if _, err := client.IM.SendStrongNotice(ctx, SendStrongNoticeOptions{
		Account: "alice", Content: "紧急通知", SMSNotice: true, CallNotice: false,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.IM.SendPhoneAlarm(ctx, SendPhoneAlarmOptions{
		Message: "支付服务", Accounts: []string{"alice"}, Mobiles: []string{"13600000000"},
	}); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(paths, []string{"/strongNotice/single/send", "/message/custom/send"}) {
		t.Fatalf("unexpected paths: %#v", paths)
	}
	if !reflect.DeepEqual(payloads[0], map[string]interface{}{
		"account": "alice", "content": "紧急通知", "sms_notice": true, "call_notice": false,
	}) {
		t.Fatalf("unexpected strong notice payload: %#v", payloads[0])
	}
	if !reflect.DeepEqual(payloads[1], map[string]interface{}{
		"tousers": []interface{}{"alice"},
		"msgtype": "voice",
		"voice":   map[string]interface{}{"mobiles": []interface{}{"13600000000"}, "message": "支付服务"},
	}) {
		t.Fatalf("unexpected phone alarm payload: %#v", payloads[1])
	}
}

func TestSendTextBuildsGroupPayload(t *testing.T) {
	t.Parallel()
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/message/custom/send" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		body, _ := ioutil.ReadAll(request.Body)
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	_, err := client.IM.SendText(context.Background(), SendIMTextOptions{To: client.To.Group("group"), Text: "hello @alice"})
	if err != nil {
		t.Fatal(err)
	}
	if payload["msgtype"] != "text" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	groups := payload["togroups"].([]interface{})
	mentions := payload["at"].([]interface{})
	if groups[0] != "group" || mentions[0] != "alice" {
		t.Fatalf("unexpected target or mentions: %#v", payload)
	}
}

func TestUploadDetectsSupportedImages(t *testing.T) {
	t.Parallel()
	if detectUploadMediaType("image/png", "image.bin") != "image" {
		t.Fatal("PNG must upload as image")
	}
	if detectUploadMediaType("image/svg+xml", "image.svg") != "file" {
		t.Fatal("SVG must upload as file")
	}
}

func TestFlattenFileSpaceListBuildsPaths(t *testing.T) {
	t.Parallel()
	items := FlattenFileSpaceList([]FileSpaceNode{
		{"node_id": "folder", "node_type": NodeTypeDir, "name": "docs"},
		{"node_id": "file", "parent_id": "folder", "node_type": NodeTypeFile, "name": "a.txt", "file_url": "url"},
	})
	if len(items) != 1 || items[0].Filename != "docs/a.txt" {
		t.Fatalf("unexpected items: %#v", items)
	}
}
