package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFileSpace删除节点发送完整参数(t *testing.T) {
	t.Parallel()
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("请求方法错误：%s", request.Method)
		}
		if request.URL.Path != "/file_space/node/delete" {
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
	_, err = client.FileSpace.DeleteNode(context.Background(), DeleteFileSpaceNodeOptions{
		SpaceID:   "team",
		SpaceType: SpaceTypeTeam,
		NodeID:    "node",
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload["space_id"] != "team" || payload["space_type"] != SpaceTypeTeam || payload["node_id"] != "node" {
		t.Fatalf("请求参数错误：%#v", payload)
	}
}
