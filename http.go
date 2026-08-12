package tuitui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type APIResponse map[string]any

type httpAPI struct{ config resolvedConfig }

func (h *httpAPI) get(ctx context.Context, endpoint string) (APIResponse, error) {
	return h.request(ctx, http.MethodGet, endpoint, nil, "")
}

func (h *httpAPI) post(ctx context.Context, endpoint string, payload any) (APIResponse, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("[tuitui] encode %s request: %w", endpoint, err)
	}
	return h.request(ctx, http.MethodPost, endpoint, bytes.NewReader(body), "application/json")
}

func (h *httpAPI) postMultipart(ctx context.Context, endpoint, filename, contentType string, data []byte) (APIResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("media", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	_ = contentType // multipart.CreateFormFile uses application/octet-stream; server detects contents.
	return h.request(ctx, http.MethodPost, endpoint, &body, writer.FormDataContentType())
}

func (h *httpAPI) request(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (APIResponse, error) {
	normalized := "/" + strings.TrimLeft(endpoint, "/")
	requestURL, err := url.Parse(h.config.apiBaseURL + normalized)
	if err != nil {
		return nil, err
	}
	query := requestURL.Query()
	query.Set("appid", h.config.appID)
	query.Set("secret", h.config.appSecret)
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, err
	}
	req.Close = true
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	started := time.Now()
	if h.config.logger != nil {
		h.config.logger.Debug("[tuitui] " + endpoint + " request")
		defer func() {
			h.config.logger.Debug(fmt.Sprintf(
				"[tuitui] %s completed in %dms",
				endpoint,
				time.Since(started).Milliseconds(),
			))
		}()
	}

	// HTTP API calls deliberately use a fresh transport and client. Keep-alive is disabled,
	// so no connection pool or connection reuse survives a request.
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: -1}).DialContext,
		ForceAttemptHTTP2:     false,
		DisableKeepAlives:     true,
		MaxIdleConns:          -1,
		IdleConnTimeout:       0,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: h.config.httpTimeout}
	defer transport.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		return nil, newAPIError(endpoint, "request failed", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, newAPIError(endpoint, "response read failed", err)
	}
	var data APIResponse
	if len(raw) == 0 {
		data = APIResponse{}
	} else if err := json.Unmarshal(raw, &data); err != nil {
		return nil, &APIError{Message: endpoint + " returned invalid JSON", Endpoint: endpoint, Status: response.StatusCode, Response: string(raw), Cause: err}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &APIError{Message: fmt.Sprintf("%s failed: %s", endpoint, response.Status), Endpoint: endpoint, Status: response.StatusCode, Response: data}
	}
	errCode := numberAsInt(data["errcode"], -1)
	if errCode != 0 {
		detail, _ := data["errmsg"].(string)
		if strings.TrimSpace(detail) == "" {
			encoded, _ := json.Marshal(data)
			detail = string(encoded)
		}
		return nil, &APIError{Message: fmt.Sprintf("%s failed with errcode %v: %s", endpoint, data["errcode"], detail), Endpoint: endpoint, ErrCode: errCode, Response: data}
	}
	return data, nil
}

func numberAsInt(value any, fallback int) int {
	switch value := value.(type) {
	case float64:
		return int(value)
	case json.Number:
		var result int
		if _, err := fmt.Sscan(string(value), &result); err == nil {
			return result
		}
	case string:
		var result int
		if _, err := fmt.Sscan(value, &result); err == nil {
			return result
		}
	case int:
		return value
	}
	return fallback
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
