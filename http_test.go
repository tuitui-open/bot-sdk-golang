package tuitui

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestHTTPRequestsUseIndependentTransports(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	connections := map[net.Conn]struct{}{}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("appid") != "app" || request.URL.Query().Get("secret") != "secret" {
			t.Fatalf("credentials missing from query: %s", request.URL.RawQuery)
		}
		if request.Close {
			t.Error("请求不应强制关闭连接")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"errcode":0}`))
	}))
	server.Config.ConnState = func(connection net.Conn, state http.ConnState) {
		if state == http.StateNew {
			mu.Lock()
			connections[connection] = struct{}{}
			mu.Unlock()
		}
	}
	server.Start()
	defer server.Close()

	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	for index := 0; index < 2; index++ {
		if _, err := client.Request(context.Background(), "/ping", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(connections) != 2 {
		t.Fatalf("两次请求应使用独立连接，实际连接数：%d", len(connections))
	}
}

func TestHTTPErrorPreservesBusinessResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 42, "errmsg": "denied", "trans_id": "tx"})
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	_, err := client.Request(context.Background(), "/failure", nil, nil)
	apiError, ok := err.(*APIError)
	if !ok || apiError.ErrCode != 42 || apiError.Response == nil {
		t.Fatalf("unexpected error: %#v", err)
	}
	want := `/failure failed with errcode 42: denied; api response: {"errcode":42,"errmsg":"denied","trans_id":"tx"}`
	if err.Error() != want {
		t.Fatalf("unexpected error text:\nwant: %s\n got: %s", want, err)
	}
}

func TestHTTPAndDecodeErrorsIncludeResponseBody(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{
			name:        "HTTP 状态错误",
			status:      http.StatusInternalServerError,
			body:        `{"error":true,"request_id":"tx"}`,
			wantMessage: `/failure failed: 500 Internal Server Error; api response: {"error":true,"request_id":"tx"}`,
		},
		{
			name:        "非法 JSON",
			status:      http.StatusOK,
			body:        `not-json`,
			wantMessage: `/failure returned invalid JSON: invalid character 'o' in literal null (expecting 'u'); api response: not-json`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
			_, err := client.Request(context.Background(), "/failure", nil, nil)
			if err == nil || err.Error() != test.wantMessage {
				t.Fatalf("unexpected error text:\nwant: %s\n got: %v", test.wantMessage, err)
			}
		})
	}
}
