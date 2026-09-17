//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func TestFile查询存在不存在和部分存在(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET")
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	ctx := context.Background()
	runID := "go-file-" + time.Now().Format("20060102150405.000000000")
	uploaded, err := client.File.Upload(ctx, []byte(runID), &tuitui.UploadOptions{Filename: runID + ".txt", ContentType: "text/plain"})
	if err != nil {
		t.Fatal(err)
	}
	missingBytes := make([]byte, 12)
	if _, err := rand.Read(missingBytes); err != nil {
		t.Fatal(err)
	}
	missingFID := hex.EncodeToString(missingBytes)

	url, err := client.File.Query(ctx, uploaded.FID)
	if err != nil || !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		t.Fatalf("unexpected query result: %q, %v", url, err)
	}
	_, err = client.File.Query(ctx, missingFID)
	apiError, ok := err.(*tuitui.APIError)
	if !ok || apiError.ErrCode != 1 || !strings.Contains(apiError.Error(), "文件不存在") {
		t.Fatalf("unexpected missing error: %#v", err)
	}
	urls, err := client.File.BatchQuery(ctx, []string{uploaded.FID, missingFID})
	if err != nil {
		t.Fatal(err)
	}
	if !(strings.HasPrefix(urls[uploaded.FID], "http://") || strings.HasPrefix(urls[uploaded.FID], "https://")) {
		t.Fatalf("unexpected uploaded URL: %q", urls[uploaded.FID])
	}
	if _, exists := urls[missingFID]; exists {
		t.Fatalf("missing fid must not be returned: %#v", urls)
	}
}
