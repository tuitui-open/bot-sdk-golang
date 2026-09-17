package tuitui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFile查询存在不存在和部分存在(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path != "/media/fetch" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("appid") != "app" || request.URL.Query().Get("secret") != "secret" {
			t.Fatalf("missing authentication query: %s", request.URL.RawQuery)
		}
		var body struct {
			MediaIDs []string `json:"media_ids"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		writer.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if !reflect.DeepEqual(body.MediaIDs, []string{"fid"}) {
				t.Fatalf("unexpected single query: %#v", body.MediaIDs)
			}
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 0, "media_url": map[string]string{"fid": "https://example.com/file"}})
		case 2:
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 1, "errmsg": "文件不存在"})
		case 3:
			if !reflect.DeepEqual(body.MediaIDs, []string{"existing", "missing"}) {
				t.Fatalf("unexpected batch query: %#v", body.MediaIDs)
			}
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"errcode": 0, "media_url": map[string]string{"existing": "https://example.com/existing"}})
		}
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL})
	ctx := context.Background()

	url, err := client.File.Query(ctx, "fid")
	if err != nil || url != "https://example.com/file" {
		t.Fatalf("unexpected query result: %q, %v", url, err)
	}
	_, err = client.File.Query(ctx, "000000000000000000000000")
	apiError, ok := err.(*APIError)
	if !ok || apiError.Endpoint != "/media/fetch" || apiError.ErrCode != 1 {
		t.Fatalf("unexpected missing error: %#v", err)
	}
	urls, err := client.File.BatchQuery(ctx, []string{"existing", "missing"})
	if err != nil || urls["existing"] != "https://example.com/existing" {
		t.Fatalf("unexpected partial result: %#v, %v", urls, err)
	}
	if _, exists := urls["missing"]; exists {
		t.Fatalf("missing fid must not be returned: %#v", urls)
	}
}
