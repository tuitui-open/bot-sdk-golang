package sample

import (
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func Test地址配置与默认值(t *testing.T) {
	names := []string{"TUITUI_BOT_API_BASE_URL", "TUITUI_BOT_WEBSOCKET_BASE_URL"}
	for _, name := range names {
		old, exists := os.LookupEnv(name)
		defer func(name, old string, exists bool) {
			if exists {
				os.Setenv(name, old)
			} else {
				os.Unsetenv(name)
			}
		}(name, old, exists)
		os.Unsetenv(name)
	}
	defaults, err := OptionsFromEnvironment()
	if err != nil || defaults.APIBaseURL != "" || defaults.WebSocketBaseURL != "" {
		t.Fatal(defaults, err)
	}
	os.Setenv(names[0], "http://localhost:8123/robot")
	os.Setenv(names[1], "ws://localhost:8123/robot")
	options, err := OptionsFromEnvironment()
	if err != nil || options.APIBaseURL != "http://localhost:8123/robot" || options.WebSocketBaseURL != "ws://localhost:8123/robot" {
		t.Fatal(options, err)
	}
	for _, value := range []string{"relative", "ftp://example.com", "https://"} {
		os.Setenv(names[0], value)
		if _, err := OptionsFromEnvironment(); err == nil {
			t.Fatal(value)
		}
	}
}

func Test配置文件保留进程变量(t *testing.T) {
	directory, err := ioutil.TempDir("", "agent-sample-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	filename := filepath.Join(directory, ".env")
	if err := ioutil.WriteFile(filename, []byte("TUITUI_BOT_API_BASE_URL=http://file.example/robot\n"), 0600); err != nil {
		t.Fatal(err)
	}
	name := "TUITUI_BOT_API_BASE_URL"
	old, exists := os.LookupEnv(name)
	defer func() {
		if exists {
			os.Setenv(name, old)
		} else {
			os.Unsetenv(name)
		}
	}()
	os.Setenv(name, "http://process.example/robot")
	if err := dotenv.Load(filename); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(name) != "http://process.example/robot" {
		t.Fatal("进程变量被覆盖")
	}
}
