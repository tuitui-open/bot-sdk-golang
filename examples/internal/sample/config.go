package sample

import (
	"fmt"
	"net/url"
	"os"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

// LoadConfig 仅供人工运行的示例加载最近的 .env，进程变量优先。
func LoadConfig() (string, string, *tuitui.ClientOptions, error) {
	if err := dotenv.LoadClosest(); err != nil {
		return "", "", nil, err
	}
	options, err := OptionsFromEnvironment()
	return os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), options, err
}
func OptionsFromEnvironment() (*tuitui.ClientOptions, error) {
	options := &tuitui.ClientOptions{APIBaseURL: os.Getenv("TUITUI_BOT_API_BASE_URL"), WebSocketBaseURL: os.Getenv("TUITUI_BOT_WEBSOCKET_BASE_URL")}
	for _, setting := range []struct {
		name, value string
		websocket   bool
	}{{"TUITUI_BOT_API_BASE_URL", options.APIBaseURL, false}, {"TUITUI_BOT_WEBSOCKET_BASE_URL", options.WebSocketBaseURL, true}} {
		if setting.value == "" {
			continue
		}
		parsed, err := url.Parse(setting.value)
		validScheme := err == nil && ((!setting.websocket && (parsed.Scheme == "http" || parsed.Scheme == "https")) || (setting.websocket && (parsed.Scheme == "ws" || parsed.Scheme == "wss")))
		if !validScheme || parsed.Host == "" {
			return nil, fmt.Errorf("%s 必须为完整的对应协议 URL", setting.name)
		}
	}
	return options, nil
}
