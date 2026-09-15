package tuitui

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

type AgentAPI struct {
	Report        *AgentReporter
	http          *httpAPI
	config        resolvedConfig
	mu            sync.Mutex
	queue         []agentJob
	consuming     bool
	lastTimestamp int64
}
type AgentReporter struct{ api *AgentAPI }

const (
	agentEventMessageReceived = "message_received"
	agentEventLLMInput        = "llm_input"
	agentEventLLMOutput       = "llm_output"
	agentEventBeforeToolCall  = "before_tool_call"
	agentEventAfterToolCall   = "after_tool_call"
	agentEventSubagentSpawned = "subagent_spawned"
	agentEventSubagentEnded   = "subagent_ended"
	agentEventEnd             = "agent_end"
)

type agentJob struct {
	ctx     context.Context
	payload map[string]interface{}
}

func agentRequired(value, field string, part bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("[tuitui] %s is required", field)
	}
	if part && strings.Contains(value, ":") {
		return "", fmt.Errorf("[tuitui] %s must not contain ':'", field)
	}
	return value, nil
}
func agentUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
func (a *AgentAPI) BuildContext(options BuildAgentContextOptions) (AgentEventContext, error) {
	result := AgentEventContext{}
	app, err := agentRequired(a.config.appID, "appId", true)
	if err != nil {
		return result, err
	}
	result.MessageID, err = agentRequired(options.MessageID, "messageId", false)
	if err != nil {
		return result, err
	}
	target := options.Target
	var route string
	switch target.Type {
	case "direct":
		route, err = agentRequired(target.Account, "target.account", true)
	case "group":
		route, err = agentRequired(target.GroupID, "target.groupId", true)
	case "channel":
		route, err = agentRequired(target.ChannelID, "target.channelId", true)
		if err != nil {
			return result, err
		}
		post, e := agentRequired(target.PostID, "target.postId", true)
		if e != nil {
			return result, e
		}
		if target.ParentPostID != nil {
			post, err = agentRequired(*target.ParentPostID, "target.parentPostId", true)
		}
		route += ":thread:" + post
	default:
		return result, fmt.Errorf("[tuitui] target.type is invalid")
	}
	if err != nil {
		return result, err
	}
	result.SessionKey = "agent:" + app + ":tuitui:" + target.Type + ":" + route
	result.RunID, err = agentUUID()
	return result, err
}
func agentString(value interface{}) string {
	valueString, _ := value.(string)
	return strings.TrimSpace(valueString)
}
func (a *AgentAPI) ContextFromMessage(event EventBody) (AgentEventContext, error) {
	data, _ := event["data"].(map[string]interface{})
	opts := BuildAgentContextOptions{MessageID: agentString(data["msgid"])}
	switch event["event"] {
	case EventSingleChat:
		opts.Target = AgentContextTarget{Type: "direct", Account: agentString(event["user_account"])}
	case EventGroupChat:
		group := agentString(data["group_id"])
		if group == "" {
			group = agentString(event["group_id"])
		}
		opts.Target = AgentContextTarget{Type: "group", GroupID: group}
	case EventTeamsPostCreate:
		post := agentString(data["post_id"])
		opts.MessageID = post
		opts.Target = AgentContextTarget{Type: "channel", ChannelID: agentString(data["channel_id"]), PostID: post}
		if data["is_reply"] == true {
			parent := agentString(data["parent_id"])
			if parent == "" || parent == "0" {
				return AgentEventContext{}, fmt.Errorf("[tuitui] teams reply data.parent_id is required")
			}
			opts.Target.ParentPostID = &parent
		}
	default:
		return AgentEventContext{}, fmt.Errorf("[tuitui] unsupported Agent message event: %v", event["event"])
	}
	return a.BuildContext(opts)
}
func agentRoute(key string, sub bool) bool {
	p := strings.Split(key, ":")
	if len(p) < 4 || p[0] != "agent" || p[1] == "" {
		return false
	}
	if sub && len(p) == 4 && p[2] == "subagent" && p[3] != "" {
		return true
	}
	if len(p) < 5 || p[2] == "" {
		return false
	}
	if p[3] == "group" && p[4] != "" {
		return true
	}
	if len(p) >= 7 && p[3] == "channel" && p[4] != "" && p[5] == "thread" && p[6] != "" {
		return true
	}
	for i := 3; i < len(p)-1; i++ {
		if p[i] == "direct" && p[i+1] != "" {
			return true
		}
	}
	return false
}
func (a *AgentAPI) BuildSubagentContext(options BuildAgentSubagentContextOptions) (AgentEventSubagentContext, error) {
	result := AgentEventSubagentContext{}
	app, err := agentRequired(a.config.appID, "appId", true)
	if err != nil {
		return result, err
	}
	parent := options.ParentContext
	parent.SessionKey, err = agentRequired(parent.SessionKey, "parentContext.sessionKey", false)
	if err != nil {
		return result, err
	}
	if !agentRoute(parent.SessionKey, false) {
		return result, fmt.Errorf("[tuitui] parentContext.sessionKey is not routable")
	}
	if _, err = agentRequired(parent.RunID, "parentContext.runId", false); err != nil {
		return result, err
	}
	result.MessageID, err = agentRequired(parent.MessageID, "parentContext.messageId", false)
	if err != nil {
		return result, err
	}
	var id string
	if options.SubagentID == nil {
		id, err = agentUUID()
	} else {
		id, err = agentRequired(*options.SubagentID, "subagentId", true)
	}
	if err != nil {
		return result, err
	}
	result.SessionKey = "agent:" + app + ":subagent:" + id
	result.RequesterSessionKey = parent.SessionKey
	result.RunID, err = agentUUID()
	return result, err
}

// agentDataFields 合并扩展字段并拒绝覆盖已声明字段，包括被省略的可选字段。
func agentDataFields(data interface{}) (map[string]interface{}, error) {
	value := reflect.ValueOf(data)
	typ := value.Type()
	result := map[string]interface{}{}
	reserved := map[string]bool{}
	var extensions map[string]interface{}
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "Extensions" {
			extensions, _ = value.Field(i).Interface().(map[string]interface{})
			continue
		}
		tag := strings.Split(field.Tag.Get("json"), ",")
		reserved[tag[0]] = true
		v := value.Field(i)
		if len(tag) > 1 && tag[1] == "omitempty" {
			switch v.Kind() {
			case reflect.String:
				if v.Len() == 0 {
					continue
				}
			case reflect.Ptr, reflect.Map, reflect.Slice:
				if v.IsNil() {
					continue
				}
			case reflect.Float32, reflect.Float64:
				if v.Float() == 0 {
					continue
				}
			}
		}
		if usage, ok := v.Interface().(AgentEventLLMUsage); ok {
			m, err := agentDataFields(usage)
			if err != nil {
				return nil, err
			}
			if _, ok := m["input"]; !ok {
				m["input"] = float64(0)
			}
			if _, ok := m["output"]; !ok {
				m["output"] = float64(0)
			}
			if _, ok := m["cacheRead"]; !ok {
				m["cacheRead"] = float64(0)
			}
			if _, ok := m["total"]; !ok {
				m["total"] = usage.Input + usage.Output
			}
			result[tag[0]] = m
		} else {
			result[tag[0]] = v.Interface()
		}
	}
	for key, v := range extensions {
		if reserved[key] {
			return nil, fmt.Errorf("[tuitui] extension conflicts with data.%s", key)
		}
		result[key] = v
	}
	return result, nil
}
func agentSerialize(name string, data interface{}) (string, error) {
	var fields map[string]interface{}
	var err error
	if name == agentEventEnd || name == agentEventMessageReceived {
		var ok bool
		fields, ok = data.(map[string]interface{})
		if !ok || fields == nil {
			return "", fmt.Errorf("[tuitui] data must be an object")
		}
	} else {
		fields, err = agentDataFields(data)
		if err != nil {
			return "", err
		}
	}
	// 序列化副本用于校验任意 JSON 值，并避免修改调用方输入。
	raw, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	var copied map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err = decoder.Decode(&copied); err != nil {
		return "", err
	}
	required := func(key string) error {
		_, err := agentRequired(agentString(copied[key]), "data."+key, false)
		return err
	}
	switch name {
	case agentEventMessageReceived:
	case agentEventLLMInput:
		err = required("model")
		if err == nil {
			err = required("prompt")
		}
	case agentEventLLMOutput:
		if _, ok := copied["assistantTexts"].([]interface{}); !ok {
			err = fmt.Errorf("[tuitui] data.assistantTexts must be an array")
		}
	case agentEventBeforeToolCall, agentEventAfterToolCall:
		err = required("toolCallId")
		if err == nil {
			err = required("toolName")
		}
	case agentEventSubagentEnded:
		if v := copied["outcome"]; v != "ok" && v != "error" && v != "cancelled" {
			err = fmt.Errorf("[tuitui] data.outcome is invalid")
		}
	}
	if err != nil {
		return "", err
	}
	raw, err = json.Marshal(copied)
	return string(raw), err
}
func agentContextFields(value interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(raw, &result)
	return result, err
}

func (r *AgentReporter) MessageReceived(ctx context.Context, eventContext AgentEventContext) {
	r.api.report(ctx, agentEventMessageReceived, eventContext, map[string]interface{}{})
}
func (r *AgentReporter) LLMInput(ctx context.Context, eventContext AgentEventWorkContext, data AgentEventLLMInputData) {
	r.api.report(ctx, agentEventLLMInput, eventContext, data)
}
func (r *AgentReporter) LLMOutput(ctx context.Context, eventContext AgentEventWorkContext, data AgentEventLLMOutputData) {
	r.api.report(ctx, agentEventLLMOutput, eventContext, data)
}
func (r *AgentReporter) BeforeToolCall(ctx context.Context, eventContext AgentEventWorkContext, data AgentEventBeforeToolCallData) {
	r.api.report(ctx, agentEventBeforeToolCall, eventContext, data)
}
func (r *AgentReporter) AfterToolCall(ctx context.Context, eventContext AgentEventWorkContext, data AgentEventAfterToolCallData) {
	r.api.report(ctx, agentEventAfterToolCall, eventContext, data)
}
func (r *AgentReporter) SubagentSpawned(ctx context.Context, eventContext AgentEventSubagentContext, data AgentEventSubagentSpawnedData) {
	r.api.report(ctx, agentEventSubagentSpawned, eventContext, data)
}
func (r *AgentReporter) SubagentEnded(ctx context.Context, eventContext AgentEventSubagentContext, data AgentEventSubagentEndedData) {
	r.api.report(ctx, agentEventSubagentEnded, eventContext, data)
}
func (r *AgentReporter) AgentEnd(ctx context.Context, eventContext AgentEventContext) {
	r.api.report(ctx, agentEventEnd, eventContext, map[string]interface{}{})
}

// report 同步校验并入队，不返回投递结果；调用方必须保持进程与 context 存活。
func (a *AgentAPI) report(ctx context.Context, name string, eventContext interface{}, data interface{}) {
	defer func() {
		if value := recover(); value != nil {
			a.log(false, name, fmt.Errorf("%v", value))
		}
	}()
	fail := func(err error) { a.log(false, name, err) }
	if ctx == nil {
		fail(fmt.Errorf("[tuitui] context is required"))
		return
	}
	fields, err := agentContextFields(eventContext)
	if err != nil {
		fail(err)
		return
	}
	serialized, err := agentSerialize(name, data)
	if err != nil {
		fail(err)
		return
	}
	app, err := agentRequired(a.config.appID, "appId", false)
	if err != nil {
		fail(err)
		return
	}
	payload := map[string]interface{}{"event": name, "plugin": "tuitui", "appId": app, "accountId": app, "ctx": fields, "data": serialized}
	a.mu.Lock()
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	if timestamp <= a.lastTimestamp {
		timestamp = a.lastTimestamp + 1
	}
	a.lastTimestamp = timestamp
	payload["timestamp"] = timestamp
	a.queue = append(a.queue, agentJob{ctx: ctx, payload: payload})
	start := !a.consuming
	a.consuming = true
	a.mu.Unlock()
	if start {
		go a.consume()
	}
}
func (a *AgentAPI) consume() {
	for {
		a.mu.Lock()
		if len(a.queue) == 0 {
			a.consuming = false
			a.queue = nil
			a.mu.Unlock()
			return
		}
		job := a.queue[0]
		a.queue[0] = agentJob{}
		a.queue = a.queue[1:]
		a.mu.Unlock()
		a.send(job)
	}
}
func (a *AgentAPI) send(job agentJob) {
	name := job.payload["event"].(string)
	defer func() {
		if v := recover(); v != nil {
			a.log(false, name, fmt.Errorf("%v", v))
		}
	}()
	response, err := a.http.post(job.ctx, "/openclaw/report", []interface{}{job.payload})
	if err != nil {
		a.log(false, name, err)
	} else {
		a.log(true, name, response)
	}
}
func (a *AgentAPI) log(success bool, event string, value interface{}) {
	if a.config.logger == nil {
		return
	}
	if success {
		a.config.logger.Info("[tuitui] Agent event reported", map[string]interface{}{"event": event, "response": value})
	} else {
		a.config.logger.Error("[tuitui] Agent event report failed", map[string]interface{}{"event": event, "error": value})
	}
}
