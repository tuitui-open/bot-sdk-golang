package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGroup按开放平台接口名称发送七种请求(t *testing.T) {
	t.Parallel()
	var paths []string
	var payloads []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if request.Method == http.MethodPost {
			var payload map[string]interface{}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("解析群管理请求失败：%v", err)
			}
			payloads = append(payloads, payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		response := `{"errcode":0,"extra":"keep"}`
		switch request.URL.Path {
		case "/group/create":
			response = `{"errcode":0,"group_id":123,"extra":"keep"}`
		case "/group/robot/in":
			response = `{"errcode":0,"groups":[{"group_id":123,"name":"机器人所在群"}]}`
		case "/group/user/isin":
			response = `{"errcode":0,"groups":[{"group_id":"789","name":"共同群"}]}`
		case "/group/members":
			response = `{"errcode":0,"members":[{"uid":123,"account":"alice","name":"Alice","role":"normal","is_bot":false,"dept":["360","研发"]},{"uid":"456","account":"bot-test","name":"Bot","role":"manager","is_bot":true,"bot_info":{"desc":"测试机器人"}}]}`
		case "/group/info":
			response = `{"errcode":0,"group_info":{"id":789,"name":"测试群","announce":"群公告"}}`
		}
		_, _ = writer.Write([]byte(response))
	}))
	defer server.Close()

	group := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL}).Group
	ctx := context.Background()
	result, err := group.Create(ctx, CreateGroupOptions{
		Name: "测试群", Owner: "alice", Members: []string{"alice", "bob"},
	})
	if err != nil || result != (CreateGroupResult{GroupID: "123"}) {
		t.Fatalf("创建群返回不正确：%#v, %v", result, err)
	}
	if _, err := group.MemberAdd(ctx, "g1", []string{"carol"}); err != nil {
		t.Fatal(err)
	}
	if _, err := group.MemberRemove(ctx, "g1", []string{"bob"}); err != nil {
		t.Fatal(err)
	}
	groups, err := group.RobotIn(ctx)
	if err != nil || !reflect.DeepEqual(groups, []GroupListItem{{GroupID: "123", Name: "机器人所在群"}}) {
		t.Fatalf("机器人所在群返回不正确：%#v, %v", groups, err)
	}
	groups, err = group.UserIsIn(ctx, UserIsInOptions{User: "alice", Groups: []string{"g1", "g2"}})
	if err != nil || !reflect.DeepEqual(groups, []GroupListItem{{GroupID: "789", Name: "共同群"}}) {
		t.Fatalf("共同群返回不正确：%#v, %v", groups, err)
	}
	members, err := group.Members(ctx, "g1")
	if err != nil || !reflect.DeepEqual(members, []GroupMember{
		{UID: "123", Account: "alice", Name: "Alice", Role: "normal", IsBot: false, Dept: []string{"360", "研发"}},
		{UID: "456", Account: "bot-test", Name: "Bot", Role: "manager", IsBot: true, BotInfo: &GroupMemberBotInfo{Desc: "测试机器人"}, Dept: []string{}},
	}) {
		t.Fatalf("群成员返回不正确：%#v, %v", members, err)
	}
	info, err := group.Info(ctx, "g1")
	if err != nil || info != (GroupInfo{ID: "789", Name: "测试群", Announce: "群公告"}) {
		t.Fatalf("群信息返回不正确：%#v, %v", info, err)
	}

	wantPaths := []string{
		"/group/create",
		"/group/member/add",
		"/group/member/remove",
		"/group/robot/in",
		"/group/user/isin",
		"/group/members",
		"/group/info",
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("群管理请求路径不正确：%#v", paths)
	}
	if !reflect.DeepEqual(payloads[0], map[string]interface{}{
		"name": "测试群", "owner": "alice", "members": []interface{}{"alice", "bob"},
	}) {
		t.Fatalf("建群请求不正确：%#v", payloads[0])
	}
	if !reflect.DeepEqual(payloads[3], map[string]interface{}{
		"user": "alice", "groups": []interface{}{"g1", "g2"},
	}) {
		t.Fatalf("查询用户所在群请求不正确：%#v", payloads[3])
	}
}
