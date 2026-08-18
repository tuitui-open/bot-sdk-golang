package tuitui

import (
	"net/http"
	"strings"
	"time"
)

const ProductionHost = "im.live.360.cn"

// Logger is the optional logging boundary used by the SDK.
type Logger interface {
	Debug(message string, context ...interface{})
	Info(message string, context ...interface{})
	Warn(message string, context ...interface{})
	Error(message string, context ...interface{})
}

// RemoteFetcher downloads HTTP/HTTPS upload sources.
type RemoteFetcher func(url string) (*http.Response, error)

type ClientOptions struct {
	APIBaseURL       string
	WebSocketBaseURL string
	// FetchWithSSRF 下载远程 HTTP/HTTPS 上传源；仅在需要自定义 SSRF 防护时设置，普通 Bot API 请求不使用。
	FetchWithSSRF RemoteFetcher
	Logger        Logger
	HTTPTimeout   time.Duration
}

type resolvedConfig struct {
	appID            string
	appSecret        string
	apiBaseURL       string
	webSocketBaseURL string
	fetchWithSSRF    RemoteFetcher
	logger           Logger
	httpTimeout      time.Duration
}

func resolveConfig(appID, appSecret string, options *ClientOptions) resolvedConfig {
	resolved := ClientOptions{}
	if options != nil {
		resolved = *options
	}
	if resolved.APIBaseURL == "" {
		resolved.APIBaseURL = "https://" + ProductionHost + ":8282/robot"
	}
	if resolved.WebSocketBaseURL == "" {
		resolved.WebSocketBaseURL = "wss://" + ProductionHost + ":8282/robot"
	}
	if resolved.HTTPTimeout <= 0 {
		resolved.HTTPTimeout = 30 * time.Second
	}
	return resolvedConfig{
		appID:            appID,
		appSecret:        appSecret,
		apiBaseURL:       strings.TrimRight(resolved.APIBaseURL, "/"),
		webSocketBaseURL: strings.TrimRight(resolved.WebSocketBaseURL, "/"),
		fetchWithSSRF:    resolved.FetchWithSSRF,
		logger:           resolved.Logger,
		httpTimeout:      resolved.HTTPTimeout,
	}
}
