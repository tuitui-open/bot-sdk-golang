//go:build e2e
// +build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

const e2eTimeoutSeconds = 15

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	rootEnv := filepath.Join(filepath.Dir(filename), "..", "..", "..", ".env")
	_ = dotenv.Load(rootEnv)
}

func requireEnv(t *testing.T, names ...string) {
	t.Helper()
	missing := make([]string, 0)
	for _, name := range names {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("请设置环境变量或填写仓库根目录 .env：%s", strings.Join(missing, ", "))
	}
}
