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
func (e *EventAPI) RenderMessageBody(data interface{}) string       { return RenderMessageBody(data) }

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
	text, err := s.teams.GetSharedPostForAgent(s.ctx, fmt.Sprint(message["shared_post"]))
	if err != nil {
		return err
	}
	message["msg_type"] = "text"
	message["text"] = text
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
	value, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	result := []MessageMedia{}
	appendURL := func(raw interface{}, mime string) {
		if text, ok := raw.(string); ok && text != "" {
			result = append(result, MessageMedia{mime, text})
		}
	}
	if images, ok := value["images"].([]interface{}); ok {
		for _, image := range images {
			appendURL(image, "image/*")
		}
	}
	appendURL(value["voice"], "audio/*")
	appendURL(value["video"], "video/*")
	if file, ok := value["file"].(map[string]interface{}); ok {
		appendURL(file["url"], "application/octet-stream")
	}
	if files, ok := value["files"].([]interface{}); ok {
		for _, raw := range files {
			if file, ok := raw.(map[string]interface{}); ok {
				appendURL(file["url"], "application/octet-stream")
			}
		}
	}
	return result
}
func RenderMessageBody(data interface{}) string {
	value, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	parts := renderBaseMessage(value)
	if ref, ok := value["ref"].(map[string]interface{}); ok {
		reference := strings.Join(renderBaseMessage(ref), "\n")
		if ref["shared_post"] != nil {
			parts = append(parts, "\n[原始背景信息参考如下]\n "+reference)
		} else {
			parts = append(parts, fmt.Sprintf("\n[引用来自 %v (%v) 的消息，内容如下]\n %s", ref["user_name"], ref["user_account"], reference))
		}
	}
	return strings.Join(parts, "\n")
}
func renderBaseMessage(data map[string]interface{}) []string {
	kind := fmt.Sprint(data["msg_type"])
	parts := []string{}
	if (kind == "text" || kind == "mixed") && fmt.Sprint(data["text"]) != "<nil>" {
		parts = append(parts, fmt.Sprint(data["text"]))
	}
	if images, ok := data["images"].([]interface{}); ok && (kind == "mixed" || kind == "image") {
		for _, image := range images {
			parts = append(parts, "[图片] "+fmt.Sprint(image))
		}
	}
	if kind == "voice" {
		parts = append(parts, "[语音] "+fmt.Sprint(data["voice"]))
	}
	if kind == "video" {
		parts = append(parts, "[视频] "+fmt.Sprint(data["video"]))
	}
	if kind == "file" {
		if file, ok := data["file"].(map[string]interface{}); ok {
			parts = append(parts, fmt.Sprintf("[文件] %v : %v", file["name"], file["url"]))
		}
	}
	if kind == "link" {
		if link, ok := data["link"].(map[string]interface{}); ok {
			parts = append(parts, fmt.Sprintf("[网页链接]\n%v\n%v", link["title"], link["url"]))
		}
	}
	return parts
}
