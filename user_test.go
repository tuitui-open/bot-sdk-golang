package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestUser原样提交关键词并保留用户字段(t *testing.T) {
	for _, keyword := range []string{"zhq", " zhq ", ""} {
		t.Run("关键词_"+keyword, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/user/search" {
					t.Errorf("请求不正确：%s %s", r.Method, r.URL.Path)
				}
				var payload map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !reflect.DeepEqual(payload, map[string]interface{}{"keyword": keyword}) {
					t.Errorf("关键词未原样提交：%v %v", payload, err)
				}
				_, _ = w.Write([]byte(`{"errcode":0,"users":[{"name":"测试用户","account":"test-user","user_id":"123","department":"研发","extra":{"enabled":true}},{"user_id":456,"extra":null}]}`))
			}))
			defer server.Close()
			client := NewClient("a", "s", &ClientOptions{APIBaseURL: server.URL})
			users, err := client.User.Search(context.Background(), keyword)
			want := []UserSearchResult{
				{"name": "测试用户", "account": "test-user", "user_id": "123", "department": "研发", "extra": map[string]interface{}{"enabled": true}},
				{"user_id": float64(456), "extra": nil},
			}
			if err != nil || !reflect.DeepEqual(users, want) {
				t.Fatalf("用户字段或顺序不正确：%v %v", users, err)
			}
		})
	}
}

func TestUser空数组与异常响应(t *testing.T) {
	for _, body := range []string{`{"errcode":0,"users":[]}`, `{"errcode":0}`, `{"errcode":0,"users":null}`, `{"errcode":0,"users":{}}`, `{"errcode":0,"users":[null]}`, `{"errcode":0,"users":["bad"]}`, `{"errcode":0,"users":[[]]}`} {
		t.Run("响应_"+body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			users, err := NewClient("a", "s", &ClientOptions{APIBaseURL: server.URL}).User.Search(context.Background(), "zhq")
			if strings.Contains(body, `"users":[]`) {
				if err != nil || users == nil || len(users) != 0 {
					t.Fatalf("应返回非 nil 空数组：%v %v", users, err)
				}
			} else {
				apiErr, ok := err.(*APIError)
				if !ok || apiErr.Endpoint != "/user/search" || apiErr.Response == nil {
					t.Fatalf("缺少协议错误详情：%v", err)
				}
			}
		})
	}
}

func TestUser业务状态与网络错误(t *testing.T) {
	for _, status := range []int{200, 503} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"errcode":403,"errmsg":"无权限"}`))
		}))
		client := NewClient("a", "s", &ClientOptions{APIBaseURL: server.URL})
		_, err := client.User.Search(context.Background(), "zhq")
		apiErr, ok := err.(*APIError)
		if !ok || apiErr.Endpoint != "/user/search" || (status == 200 && (!strings.Contains(err.Error(), "无权限") || apiErr.ErrCode != 403)) {
			t.Fatalf("错误详情缺失：%v", err)
		}
		server.Close()
		if _, err := client.User.Search(context.Background(), "zhq"); err == nil {
			t.Fatal("网络错误不应转换为成功")
		}
	}
}

func TestUser隔离实例凭证并支持取消(t *testing.T) {
	clients := make([]*Client, 0, 2)
	for _, app := range []string{"a", "b"} {
		app := app
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+app+"-secret-"+app || r.URL.RawQuery != "" {
				t.Error("实例凭证混用")
			}
			_, _ = w.Write([]byte(`{"errcode":0,"users":[]}`))
		}))
		defer server.Close()
		clients = append(clients, NewClient(app, "secret-"+app, &ClientOptions{APIBaseURL: server.URL}))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := clients[0].User.Search(ctx, "zhq")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("取消错误未保留：%v", err)
	}
	urlErr, ok := apiErr.Cause.(*url.Error)
	if !ok || urlErr.Err != context.Canceled {
		t.Fatalf("取消原因未保留：%v", err)
	}
	for _, client := range clients {
		if _, err := client.User.Search(context.Background(), "zhq"); err != nil {
			t.Fatal(err)
		}
	}
}
