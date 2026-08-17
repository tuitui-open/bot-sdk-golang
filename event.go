package tuitui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	EventSingleChatOpen     = "single_chat_open"
	EventSingleChat         = "single_chat"
	EventGroupChat          = "group_chat"
	EventGroupCreate        = "group_create"
	EventGroupInvite        = "group_invite"
	EventGroupKick          = "group_kick"
	EventTeamsPostCreate    = "teams_post_create"
	EventTeamsPostModify    = "teams_post_modify"
	EventTeamsChannelCreate = "teams_channel_create"
	EventTeamsTeamUpdate    = "teams_team_update"
	EventTeamsTeamDelete    = "teams_team_delete"
	EventTeamsMemberAdd     = "teams_member_add"
	EventTeamsMemberRemove  = "teams_member_remove"
	EventTeamsMemberSet     = "teams_member_set"
	EventInteractiveAction  = "interactive_action"
)

type EventBody map[string]interface{}

// BotName 返回接收该事件的机器人名称，始终返回字符串。
func (body EventBody) BotName() string {
	name, _ := body["bot_name"].(string)
	return name
}

type eventEnvelope struct {
	EventID string                 `json:"event_id"`
	Header  map[string]interface{} `json:"header"`
	Body    EventBody              `json:"body"`
}
type MessageMedia struct{ MIMEType, URL string }

// DetailedMessageMedia 是保留文件名称和文件标识的消息媒体。
type DetailedMessageMedia struct{ MIMEType, URL, Name, FileID string }

const normalizedMessageMediaKey = "_tuitui_media"

type SubscribeOptions struct {
	OnEvent           func(EventBody)
	OnConnected       func()
	OnDisconnected    func(error)
	OnError           func(error)
	ReconnectDelay    time.Duration
	MaxReconnectDelay time.Duration
	ConnectionTimeout time.Duration
	HeartbeatTimeout  time.Duration
	DeduplicationSize int
	DeduplicationTTL  time.Duration
}
type EventAPI struct {
	config resolvedConfig
	teams  *TeamsAPI
}

func (e *EventAPI) Subscribe(ctx context.Context, options *SubscribeOptions) *Subscription {
	resolved := SubscribeOptions{ReconnectDelay: time.Second, MaxReconnectDelay: time.Minute, ConnectionTimeout: 30 * time.Second, HeartbeatTimeout: time.Minute, DeduplicationSize: 1000, DeduplicationTTL: 20 * time.Minute}
	if options != nil {
		resolved = *options
		if resolved.ReconnectDelay <= 0 {
			resolved.ReconnectDelay = time.Second
		}
		if resolved.MaxReconnectDelay <= 0 {
			resolved.MaxReconnectDelay = time.Minute
		}
		if resolved.ConnectionTimeout <= 0 {
			resolved.ConnectionTimeout = 30 * time.Second
		}
		if resolved.HeartbeatTimeout <= 0 {
			resolved.HeartbeatTimeout = time.Minute
		}
		if resolved.DeduplicationSize <= 0 {
			resolved.DeduplicationSize = 1000
		}
		if resolved.DeduplicationTTL <= 0 {
			resolved.DeduplicationTTL = 20 * time.Minute
		}
	}
	child, cancel := context.WithCancel(ctx)
	subscription := &Subscription{ctx: child, cancel: cancel, config: e.config, teams: e.teams, options: resolved, done: make(chan struct{}), seen: map[string]time.Time{}}
	go subscription.run()
	return subscription
}
func (e *EventAPI) GetMessageMedia(data interface{}) []MessageMedia { return GetMessageMedia(data) }

// GetDetailedMessageMedia 返回包含名称和文件标识的消息媒体。
func (e *EventAPI) GetDetailedMessageMedia(data interface{}) []DetailedMessageMedia {
	return GetDetailedMessageMedia(data)
}
func (e *EventAPI) RenderMessageBody(data interface{}) string { return RenderMessageBody(data) }

type Subscription struct {
	ctx     context.Context
	cancel  context.CancelFunc
	config  resolvedConfig
	teams   *TeamsAPI
	options SubscribeOptions
	done    chan struct{}
	mu      sync.Mutex
	conn    *websocket.Conn
	seen    map[string]time.Time
}

func (s *Subscription) Unsubscribe() {
	s.cancel()
	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.mu.Unlock()
	<-s.done
}
func (s *Subscription) run() {
	defer close(s.done)
	attempt := 0
	for {
		s.selectDone()
		if s.ctx.Err() != nil {
			return
		}
		connection, _, err := websocket.DefaultDialer.DialContext(s.ctx, s.websocketURL(), nil)
		if err != nil {
			s.report(err)
			if !s.wait(attempt) {
				return
			}
			attempt++
			continue
		}
		s.mu.Lock()
		s.conn = connection
		s.mu.Unlock()
		attempt = 0
		if s.options.OnConnected != nil {
			s.options.OnConnected()
		}
		err = s.receive(connection)
		s.mu.Lock()
		if s.conn == connection {
			s.conn = nil
		}
		s.mu.Unlock()
		_ = connection.Close()
		if s.ctx.Err() != nil {
			return
		}
		if s.options.OnDisconnected != nil {
			s.options.OnDisconnected(err)
		}
		if !s.wait(attempt) {
			return
		}
		attempt++
	}
}
func (s *Subscription) selectDone() {
	select {
	case <-s.ctx.Done():
		return
	default:
	}
}
func (s *Subscription) websocketURL() string {
	query := url.Values{}
	query.Set("auth", s.config.appID+"."+s.config.appSecret)
	return s.config.webSocketBaseURL + "/callback/ws?" + query.Encode()
}
func (s *Subscription) receive(connection *websocket.Conn) error {
	for {
		if err := connection.SetReadDeadline(time.Now().Add(s.options.HeartbeatTimeout)); err != nil {
			return err
		}
		_, data, err := connection.ReadMessage()
		if err != nil {
			return err
		}
		var event eventEnvelope
		if err := json.Unmarshal(data, &event); err != nil {
			s.report(err)
			continue
		}
		eventID := event.EventID
		if eventID == "" {
			s.report(fmt.Errorf("[tuitui] WebSocket event missing event_id"))
			continue
		}
		ack, _ := json.Marshal(map[string]string{"ack": eventID})
		if err := connection.WriteMessage(websocket.TextMessage, ack); err != nil {
			s.report(err)
			return err
		}
		if !s.record(eventID) {
			continue
		}
		body := event.Body
		if body["event"] == "keepalive" {
			continue
		}
		if appID, ok := event.Header["X-Tuitui-Robot-Appid"].(string); ok && appID != s.config.appID {
			s.report(fmt.Errorf("[tuitui] event app_id does not match client app_id"))
			continue
		}
		if body == nil {
			s.report(fmt.Errorf("[tuitui] WebSocket event missing body"))
			continue
		}
		botName, _ := event.Header["X-Tuitui-Robot-AppName"].(string)
		body["bot_name"] = botName
		if err := s.normalizeMessage(body); err != nil {
			s.report(err)
			continue
		}
		if s.options.OnEvent != nil {
			s.options.OnEvent(body)
		}
	}
}
func (s *Subscription) normalizeMessage(body EventBody) error {
	kind, _ := body["event"].(string)
	if kind != EventSingleChat && kind != EventGroupChat {
		return nil
	}
	data, _ := body["data"].(map[string]interface{})
	if data == nil {
		return nil
	}
	if err := s.normalizeSharedPost(data); err != nil {
		return err
	}
	if ref, ok := data["ref"].(map[string]interface{}); ok {
		if err := s.normalizeSharedPost(ref); err != nil {
			return err
		}
	}
	delete(data, "timestamp")
	for _, key := range []string{"msgtype", "text", "trigger_word", "images", "file"} {
		delete(body, key)
	}
	return nil
}
func (s *Subscription) normalizeSharedPost(message map[string]interface{}) error {
	if message["msg_type"] != "shared_post" {
		return nil
	}
	content, err := s.teams.getSharedPostForAgent(s.ctx, fmt.Sprint(message["shared_post"]))
	if err != nil {
		return err
	}
	message["msg_type"] = "text"
	message["text"] = content.Text
	appendDetailedMedia(message, content.Media)
	return nil
}
func (s *Subscription) record(id string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, at := range s.seen {
		if now.Sub(at) > s.options.DeduplicationTTL {
			delete(s.seen, key)
		}
	}
	if _, exists := s.seen[id]; exists {
		return false
	}
	if len(s.seen) >= s.options.DeduplicationSize {
		var oldestKey string
		var oldest time.Time
		for key, at := range s.seen {
			if oldestKey == "" || at.Before(oldest) {
				oldestKey = key
				oldest = at
			}
		}
		delete(s.seen, oldestKey)
	}
	s.seen[id] = now
	return true
}
func (s *Subscription) wait(attempt int) bool {
	delay := s.options.ReconnectDelay
	for index := 0; index < attempt && delay < s.options.MaxReconnectDelay; index++ {
		delay *= 2
	}
	if delay > s.options.MaxReconnectDelay {
		delay = s.options.MaxReconnectDelay
	}
	select {
	case <-s.ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}
func (s *Subscription) report(err error) {
	if s.options.OnError != nil {
		s.options.OnError(err)
	}
}

func GetMessageMedia(data interface{}) []MessageMedia {
	detailed := GetDetailedMessageMedia(data)
	result := make([]MessageMedia, 0, len(detailed))
	for _, media := range detailed {
		result = append(result, MessageMedia{MIMEType: media.MIMEType, URL: media.URL})
	}
	return result
}

// GetDetailedMessageMedia 递归提取正文、引用和合并转发中的媒体。
func GetDetailedMessageMedia(data interface{}) []DetailedMessageMedia {
	value, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	result := []DetailedMessageMedia{}
	seen := map[string]int{}
	collectDetailedMedia(value, &result, seen)
	return result
}

func collectDetailedMedia(data map[string]interface{}, result *[]DetailedMessageMedia, seen map[string]int) {
	if media, ok := data[normalizedMessageMediaKey].([]interface{}); ok {
		for _, item := range media {
			collectOneDetailedMedia(item, "application/octet-stream", result, seen)
		}
	}
	if images, ok := data["images"].([]interface{}); ok {
		for _, image := range images {
			if _, ok := image.(string); ok {
				collectOneDetailedMedia(image, "image/*", result, seen)
			}
		}
	}
	collectOneDetailedMedia(data["voice"], "audio/*", result, seen)
	collectOneDetailedMedia(data["video"], "video/*", result, seen)
	collectOneDetailedMedia(data["file"], "application/octet-stream", result, seen)
	if merged, ok := data["merged"].(map[string]interface{}); ok {
		if messages, ok := merged["msgs"].([]interface{}); ok {
			for _, raw := range messages {
				if message, ok := raw.(map[string]interface{}); ok {
					collectDetailedMedia(message, result, seen)
				}
			}
		}
	}
	if ref, ok := data["ref"].(map[string]interface{}); ok {
		collectDetailedMedia(ref, result, seen)
	}
}

func collectOneDetailedMedia(raw interface{}, mimeType string, result *[]DetailedMessageMedia, seen map[string]int) {
	media := detailedMedia(raw, mimeType)
	if media.URL == "" {
		return
	}
	if index, exists := seen[media.URL]; exists {
		existing := &(*result)[index]
		existing.MIMEType = firstNonEmpty(existing.MIMEType, media.MIMEType)
		existing.Name = firstNonEmpty(existing.Name, media.Name)
		existing.FileID = firstNonEmpty(existing.FileID, media.FileID)
		return
	}
	seen[media.URL] = len(*result)
	*result = append(*result, media)
}

func detailedMedia(raw interface{}, defaultMIMEType string) DetailedMessageMedia {
	if url, ok := raw.(string); ok {
		return DetailedMessageMedia{MIMEType: defaultMIMEType, URL: url}
	}
	value, _ := raw.(map[string]interface{})
	if value == nil {
		return DetailedMessageMedia{}
	}
	return DetailedMessageMedia{
		MIMEType: firstNonEmpty(stringValue(value["mime_type"]), defaultMIMEType),
		URL:      stringValue(value["url"]),
		Name:     stringValue(value["name"]),
		FileID:   firstNonEmpty(stringValue(value["file_id"]), stringValue(value["fid"])),
	}
}

func appendDetailedMedia(data map[string]interface{}, media []DetailedMessageMedia) {
	values := make([]interface{}, 0, len(media))
	for _, item := range media {
		value := map[string]interface{}{"mime_type": item.MIMEType, "url": item.URL}
		if item.Name != "" {
			value["name"] = item.Name
		}
		if item.FileID != "" {
			value["file_id"] = item.FileID
		}
		values = append(values, value)
	}
	if len(values) > 0 {
		data[normalizedMessageMediaKey] = values
	}
}

func RenderMessageBody(data interface{}) string {
	value, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	return strings.Join(renderMessage(value), "\n")
}

func renderMessage(data map[string]interface{}) []string {
	parts := renderBaseMessage(data)
	if ref, ok := data["ref"].(map[string]interface{}); ok {
		reference := strings.Join(renderMessage(ref), "\n")
		if ref["shared_post"] != nil {
			parts = append(parts, "\n[原始背景信息参考如下]\n"+reference)
		} else {
			parts = append(parts, fmt.Sprintf("\n[引用来自 %s 的消息，内容如下]\n%s", renderSender(ref), reference))
		}
	}
	return parts
}

func renderBaseMessage(data map[string]interface{}) []string {
	parts := []string{}
	if text := stringValue(data["text"]); text != "" {
		parts = append(parts, text)
	}
	seen := map[string]int{}
	appendMediaLine := func(label string, raw interface{}) {
		line, url := renderMediaLine(label, raw)
		if line == "" {
			return
		}
		if index, exists := seen[url]; exists {
			if detailedMedia(raw, "").Name != "" {
				parts[index] = line
			}
			return
		}
		seen[url] = len(parts)
		parts = append(parts, line)
	}
	if images, ok := data["images"].([]interface{}); ok {
		for _, image := range images {
			if _, ok := image.(string); ok {
				appendMediaLine("图片", image)
			}
		}
	}
	for _, item := range []struct {
		label string
		value interface{}
	}{
		{"语音", data["voice"]},
		{"视频", data["video"]},
		{"文件", data["file"]},
	} {
		appendMediaLine(item.label, item.value)
	}
	if card, ok := data["card"].(map[string]interface{}); ok {
		parts = append(parts, fmt.Sprintf("[名片]\n姓名: %s\n推推账号: %s", stringValue(card["name"]), stringValue(card["account"])))
	}
	if link, ok := data["link"].(map[string]interface{}); ok {
		parts = append(parts, fmt.Sprintf("[网页链接]\n%s\n%s", stringValue(link["title"]), stringValue(link["url"])))
	}
	if merged, ok := data["merged"].(map[string]interface{}); ok {
		parts = append(parts, renderMergedMessage(merged))
	}
	return parts
}

func renderMediaLine(label string, raw interface{}) (string, string) {
	media := detailedMedia(raw, "")
	if media.URL == "" {
		return "", ""
	}
	if media.Name != "" {
		return fmt.Sprintf("[%s] %s: %s", label, media.Name, media.URL), media.URL
	}
	return fmt.Sprintf("[%s] %s", label, media.URL), media.URL
}

func renderMergedMessage(merged map[string]interface{}) string {
	source := firstNonEmpty(stringValue(merged["source"]), "聊天记录")
	lines := []string{fmt.Sprintf("[合并转发：%s]", source)}
	messages, _ := merged["msgs"].([]interface{})
	for _, raw := range messages {
		message, _ := raw.(map[string]interface{})
		if message == nil {
			continue
		}
		lines = append(lines, "------")
		if timestamp := formatTimestamp(message["timestamp"]); timestamp != "" {
			lines = append(lines, "时间: "+timestamp)
		}
		lines = append(lines, "发言人: "+renderSender(message), "内容：")
		lines = append(lines, renderMessage(message)...)
	}
	return strings.Join(lines, "\n")
}

func renderSender(data map[string]interface{}) string {
	name := stringValue(data["user_name"])
	account := stringValue(data["user_account"])
	if name == "" {
		return firstNonEmpty(account, "unknown")
	}
	if account == "" || account == name {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, account)
}
