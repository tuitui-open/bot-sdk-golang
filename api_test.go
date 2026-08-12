package tuitui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendTextBuildsGroupPayload(t *testing.T) {
	t.Parallel()
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/message/custom/send" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()
	client, _ := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	_, err := client.IM.SendText(context.Background(), SendIMTextOptions{To: client.To.Group("group"), Text: "hello @alice"})
	if err != nil {
		t.Fatal(err)
	}
	if payload["msgtype"] != "text" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	groups := payload["togroups"].([]any)
	mentions := payload["at"].([]any)
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
