package tuitui

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestFile查询存在不存在和部分存在(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path != "/media/fetch" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer app-secret" || request.URL.RawQuery != "" {
			t.Error("文件请求鉴权应仅通过 Header 传递")
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

func TestFile远程下载不带凭证且上传使用Header(t *testing.T) {
	downloaded := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer app-secret" || r.URL.RawQuery != "type=file" {
			t.Error("上传鉴权或业务查询参数错误")
		}
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Error(err)
			return
		}
		file, header, err := r.FormFile("media")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		body, err := ioutil.ReadAll(file)
		if err != nil || string(body) != "x" || header.Filename != "x.txt" {
			t.Error("文件字段被改写")
		}
		_, _ = w.Write([]byte(`{"errcode":0,"media_id":"fid"}`))
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{
		APIBaseURL: server.URL,
		FetchWithSSRF: func(url string) (*http.Response, error) {
			if url != "https://source.test/file" {
				t.Error("远程来源被改写")
			}
			downloaded = true
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: ioutil.NopCloser(strings.NewReader("x"))}, nil
		},
	})
	result, err := client.File.Upload(context.Background(), "https://source.test/file", &UploadOptions{Filename: "x.txt"})
	if err != nil || result.FID != "fid" || !downloaded {
		t.Fatalf("远程上传失败：%v", err)
	}
}
