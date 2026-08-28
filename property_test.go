package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestProperty快捷指令Tag往返(t *testing.T) {
	t.Parallel()

	command := ShortcutCommand{
		Name:        "skill",
		Content:     "/skill",
		Description: "运行 Skill",
		Tag:         "golang",
	}
	var setPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/shortcutCommand/set":
			if err := json.NewDecoder(request.Body).Decode(&setPayload); err != nil {
				t.Errorf("解析设置快捷指令请求失败：%v", err)
			}
			_, _ = writer.Write([]byte(`{"errcode":0}`))
		case "/shortcutCommand/get":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{
				"errcode": 0,
				"datas": map[string]interface{}{
					"shortcut_cmds": []interface{}{map[string]interface{}{
						"command_name":        command.Name,
						"command_content":     command.Content,
						"command_description": command.Description,
						"tag":                 command.Tag,
					}},
				},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	if _, err := client.Property.SetShortcutCommands(context.Background(), []ShortcutCommand{command}, nil); err != nil {
		t.Fatal(err)
	}
	items, ok := setPayload["shortcut_cmds"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("设置快捷指令请求不正确：%#v", setPayload)
	}
	item, ok := items[0].(map[string]interface{})
	if !ok || item["tag"] != command.Tag {
		t.Fatalf("设置快捷指令请求缺少 tag：%#v", setPayload)
	}

	commands, err := client.Property.GetShortcutCommands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(commands, []ShortcutCommand{command}) {
		t.Fatalf("获取快捷指令未保留 tag：%#v", commands)
	}
}

func TestProperty全量设置和清空工作区菜单(t *testing.T) {
	t.Parallel()

	var payloads []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/workspaceMenu/set" {
			http.NotFound(writer, request)
			return
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("解析设置工作区菜单请求失败：%v", err)
		}
		payloads = append(payloads, payload)
		_, _ = writer.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()

	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	menus := []WorkspaceMenu{
		{Name: "终端", URL: "https://example.com/tty", OpenMode: "side_panel"},
		{Name: "应用", URL: "https://example.com/app", OpenMode: "single_tab", AppID: "example-app"},
	}
	if _, err := client.Property.SetWorkspaceMenus(context.Background(), menus); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Property.SetWorkspaceMenus(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	want := []map[string]interface{}{
		{"workspace_menus": []interface{}{
			map[string]interface{}{"name": "终端", "url": "https://example.com/tty", "open_mode": "side_panel"},
			map[string]interface{}{"name": "应用", "url": "https://example.com/app", "open_mode": "single_tab", "app_id": "example-app"},
		}},
		{"workspace_menus": []interface{}{}},
	}
	if !reflect.DeepEqual(payloads, want) {
		t.Fatalf("设置工作区菜单请求不正确：%#v", payloads)
	}
}
