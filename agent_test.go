package tuitui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type agentTestLog struct {
	info        chan map[string]interface{}
	failures    chan map[string]interface{}
	panicDebug  bool
	panicResult bool
}

func (l *agentTestLog) Debug(string, ...interface{}) {
	if l.panicDebug {
		panic("调试日志失败")
	}
}
func (l *agentTestLog) Warn(string, ...interface{}) {}
func (l *agentTestLog) Info(_ string, v ...interface{}) {
	if l.panicResult {
		panic("成功日志失败")
	}
	l.info <- v[0].(map[string]interface{})
}
func (l *agentTestLog) Error(_ string, v ...interface{}) {
	if l.panicResult {
		panic("失败日志失败")
	}
	l.failures <- v[0].(map[string]interface{})
}
func agentLogger() *agentTestLog {
	return &agentTestLog{info: make(chan map[string]interface{}, 1000), failures: make(chan map[string]interface{}, 1000)}
}
func agentWait(t *testing.T, ch <-chan map[string]interface{}) map[string]interface{} {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("等待上报超时")
		return nil
	}
}
func agentBase() AgentEventContext {
	return AgentEventContext{SessionKey: "agent:app:tuitui:group:g", MessageID: "m", RunID: "r"}
}
func agentEvent() AgentMessageReceivedEvent {
	return AgentMessageReceivedEvent{Context: agentBase(), Data: AgentEventMessageReceivedData{Content: "你好"}}
}
func TestAgent上下文路由与关联(t *testing.T) {
	a := NewClient(" app ", "secret", nil).Agent
	cases := []struct {
		name  string
		body  EventBody
		route string
	}{
		{"单聊", EventBody{"event": EventSingleChat, "user_account": " alice ", "data": map[string]interface{}{"msgid": "m"}}, "agent:app:tuitui:direct:alice"},
		{"群优先", EventBody{"event": EventGroupChat, "group_id": "fallback", "data": map[string]interface{}{"msgid": "m", "group_id": "g"}}, "agent:app:tuitui:group:g"},
		{"群回退", EventBody{"event": EventGroupChat, "group_id": "g", "data": map[string]interface{}{"msgid": "m"}}, "agent:app:tuitui:group:g"},
		{"主帖", EventBody{"event": EventTeamsPostCreate, "data": map[string]interface{}{"post_id": "m", "channel_id": "c", "is_reply": "true", "parent_id": "p"}}, "agent:app:tuitui:channel:c:thread:m"},
		{"回帖", EventBody{"event": EventTeamsPostCreate, "data": map[string]interface{}{"post_id": "m", "channel_id": "c", "is_reply": true, "parent_id": "p"}}, "agent:app:tuitui:channel:c:thread:p"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			v, err := a.ContextFromMessage(tt.body, nil)
			if err != nil || v.SessionKey != tt.route || v.MessageID != "m" || len(v.RunID) != 36 {
				t.Fatalf("%+v %v", v, err)
			}
			sub, err := a.BuildSubagentContext(BuildAgentSubagentContextOptions{RequesterContext: v})
			if err != nil || sub.RequesterSessionKey != v.SessionKey || sub.MessageID != v.MessageID || sub.RunID == v.RunID {
				t.Fatalf("%+v %v", sub, err)
			}
			fields, err := agentContextFields(AgentEventToolContext{Context: sub, ToolCallID: "tool"})
			if err != nil || fields["requesterSessionKey"] != v.SessionKey {
				t.Fatal(fields, err)
			}
		})
	}
	for _, body := range []EventBody{{"event": "unknown"}, {"event": EventSingleChat, "user_account": "a:b", "data": map[string]interface{}{"msgid": "m"}}, {"event": EventTeamsPostCreate, "data": map[string]interface{}{"post_id": "m", "channel_id": "c", "is_reply": true, "parent_id": "0"}}} {
		if _, err := a.ContextFromMessage(body, nil); err == nil {
			t.Fatal("非法上下文未拒绝")
		}
	}
	empty := ""
	if _, err := a.BuildSubagentContext(BuildAgentSubagentContextOptions{RequesterContext: agentBase(), SubagentID: &empty}); err == nil {
		t.Fatal("空子 ID 未拒绝")
	}
}
func TestAgent八种事件与裁剪(t *testing.T) {
	long := strings.Repeat("😀", 501)
	base := agentBase()
	sub := AgentEventSubagentContext{AgentEventContext: AgentEventContext{SessionKey: "agent:app:subagent:s", MessageID: "m", RunID: "s"}, RequesterSessionKey: base.SessionKey}
	input := AgentEventLLMInputData{Model: "model", Prompt: long, SystemPrompt: long, HistoryMessages: []AgentEventMessage{{"content": long, "extra": long}}, Extensions: map[string]interface{}{"message": map[string]interface{}{"content": long}, "unknown": long}}
	events := []AgentEvent{agentEvent(), AgentLLMInputEvent{Context: base, Data: input}, AgentLLMOutputEvent{Context: sub, Data: AgentEventLLMOutputData{AssistantTexts: []string{long}, Thinking: nil, LastAssistant: AgentEventMessage{"content": long}}}, AgentBeforeToolCallEvent{Context: AgentEventToolContext{Context: sub, ToolCallID: "t"}, Data: AgentEventBeforeToolCallData{ToolName: "tool"}}, AgentAfterToolCallEvent{Context: AgentEventToolContext{Context: sub, ToolCallID: "t"}, Data: AgentEventAfterToolCallData{ToolName: "tool", Result: long}}, AgentSubagentSpawnedEvent{Context: sub}, AgentSubagentEndedEvent{Context: sub, Data: AgentEventSubagentEndedData{Outcome: AgentSubagentOK}}, AgentEndEvent{Context: base, Data: AgentEventEndData{Status: AgentEndDone, Messages: []AgentEventMessage{{"content": long}}}}}
	for _, event := range events {
		name, _, data := agentFields(event)
		t.Run(name, func(t *testing.T) {
			raw, err := agentSerialize(name, data)
			if err != nil {
				t.Fatal(err)
			}
			var v map[string]interface{}
			json.Unmarshal([]byte(raw), &v)
			if name == AgentEventLLMInput {
				if v["prompt"] != long || v["unknown"] != long || !strings.HasSuffix(v["systemPrompt"].(string), "…[truncated 1 chars]") {
					t.Fatal(v)
				}
			}
			if name == AgentEventLLMOutput && v["thinking"] != nil {
				t.Fatal(v)
			}
		})
	}
	if input.HistoryMessages[0]["content"] != long || input.SystemPrompt != long {
		t.Fatal("修改了输入")
	}
	for _, v := range []interface{}{nil, false, 0, []interface{}{}, map[string]interface{}{"content": long}} {
		raw, err := agentSerialize(AgentEventAfterToolCall, AgentEventAfterToolCallData{ToolName: "tool", Result: v})
		if err != nil || !strings.Contains(raw, "\"result\":") {
			t.Fatal(raw, err)
		}
	}
	if agentTruncate(strings.Repeat("😀", 500)) != strings.Repeat("😀", 500) {
		t.Fatal("边界裁剪")
	}
}
func TestAgent非法输入仅记录失败(t *testing.T) {
	logger := agentLogger()
	a := NewClient("app", "secret", &ClientOptions{Logger: logger}).Agent
	cycle := map[string]interface{}{}
	cycle["self"] = cycle
	events := []AgentEvent{nil, AgentMessageReceivedEvent{}, AgentMessageReceivedEvent{Context: agentBase(), Data: AgentEventMessageReceivedData{Content: "x", Extensions: map[string]interface{}{"content": "y"}}}, AgentMessageReceivedEvent{Context: agentBase(), Data: AgentEventMessageReceivedData{Content: "x", Extensions: cycle}}, AgentLLMInputEvent{Context: agentBase()}, AgentLLMOutputEvent{Context: agentBase(), Data: AgentEventLLMOutputData{AssistantTexts: []string{}, Usage: AgentEventLLMUsage{Input: math.Inf(1)}}}, AgentLLMOutputEvent{Context: agentBase()}, AgentBeforeToolCallEvent{Context: AgentEventToolContext{Context: agentBase()}}, AgentEndEvent{Context: agentBase()}, AgentSubagentEndedEvent{}}
	for _, e := range events {
		a.Report(context.Background(), e)
		agentWait(t, logger.failures)
	}
	a.Report(nil, agentEvent())
	agentWait(t, logger.failures)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.consuming || len(a.queue) != 0 {
		t.Fatal("非法事件进入队列")
	}
}
func TestAgent串行取消日志与空闲重启(t *testing.T) {
	entered := make(chan map[string]interface{}, 10)
	release := make(chan struct{})
	var count int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		var payload []map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		entered <- payload[0]
		if count == 1 {
			<-release
		}
		fmt.Fprint(w, `{"errcode":0,"trans_id":"id","extra":{"kept":true}}`)
	}))
	defer server.Close()
	logger := agentLogger()
	logger.panicDebug = true
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL, Logger: logger})
	client.Agent.Report(context.Background(), agentEvent())
	first := agentWait(t, entered)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	client.Agent.Report(canceled, agentEvent())
	client.Agent.Report(context.Background(), agentEvent())
	select {
	case <-entered:
		t.Fatal("第一条未结束就发送后续")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	result := agentWait(t, logger.info)
	if result["response"].(APIResponse)["extra"] == nil {
		t.Fatal(result)
	}
	failure := agentWait(t, logger.failures)
	if strings.Contains(fmt.Sprint(failure), "secret=") {
		t.Fatal("泄漏鉴权 URL")
	}
	second := agentWait(t, entered)
	agentWait(t, logger.info)
	if first["timestamp"].(float64) >= second["timestamp"].(float64) {
		t.Fatal("时间戳未递增")
	}
	for {
		client.Agent.mu.Lock()
		idle := !client.Agent.consuming
		client.Agent.mu.Unlock()
		if idle {
			break
		}
		time.Sleep(time.Millisecond)
	}
	client.Agent.mu.Lock()
	client.Agent.lastTimestamp = time.Now().Add(time.Hour).UnixNano() / int64(time.Millisecond)
	previous := client.Agent.lastTimestamp
	client.Agent.mu.Unlock()
	client.Agent.Report(context.Background(), agentEvent())
	third := agentWait(t, entered)
	agentWait(t, logger.info)
	if int64(third["timestamp"].(float64)) != previous+1 {
		t.Fatal("时钟回拨")
	}
}
func TestAgent并发入队与实例隔离(t *testing.T) {
	var mu sync.Mutex
	timestamps := []float64{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p []map[string]interface{}
		json.NewDecoder(r.Body).Decode(&p)
		mu.Lock()
		timestamps = append(timestamps, p[0]["timestamp"].(float64))
		mu.Unlock()
		fmt.Fprint(w, `{"errcode":0}`)
	}))
	defer server.Close()
	logger := agentLogger()
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL, Logger: logger})
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); client.Agent.Report(context.Background(), agentEvent()) }()
	}
	wg.Wait()
	for i := 0; i < 80; i++ {
		agentWait(t, logger.info)
	}
	mu.Lock()
	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] <= timestamps[i-1] {
			t.Fatal("时间戳顺序错误")
		}
	}
	mu.Unlock()
	block := make(chan struct{})
	started := make(chan map[string]interface{}, 1)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { started <- nil; <-block; fmt.Fprint(w, `{"errcode":0}`) }))
	defer slow.Close()
	other := NewClient("other", "secret", &ClientOptions{APIBaseURL: slow.URL})
	other.Agent.Report(context.Background(), agentEvent())
	agentWait(t, started)
	client.Agent.Report(context.Background(), agentEvent())
	agentWait(t, logger.info)
	close(block)
}
func TestAgentHTTP失败与日志恐慌继续(t *testing.T) {
	for _, body := range []string{`{"errcode":7,"errmsg":"denied","reason":"detail"}`, `invalid`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer server.Close()
			logger := agentLogger()
			logger.panicDebug = true
			client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL, Logger: logger})
			client.Agent.Report(context.Background(), agentEvent())
			client.Agent.Report(context.Background(), agentEvent())
			for i := 0; i < 2; i++ {
				v := agentWait(t, logger.failures)
				if !strings.Contains(fmt.Sprint(v), "/openclaw/report") {
					t.Fatal(v)
				}
			}
		})
	}
	received := make(chan map[string]interface{}, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- nil
		w.WriteHeader(503)
		fmt.Fprint(w, `{"errcode":9}`)
	}))
	defer server.Close()
	logger := agentLogger()
	logger.panicResult = true
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL, Logger: logger})
	client.Agent.Report(context.Background(), agentEvent())
	client.Agent.Report(context.Background(), agentEvent())
	agentWait(t, received)
	agentWait(t, received)
}

func TestAgent保留空数组数值与扩展冲突(t *testing.T) {
	zero := float64(0)
	raw, err := agentSerialize(AgentEventLLMInput, AgentEventLLMInputData{Model: "m", Prompt: "p", HistoryMessages: []AgentEventMessage{}, ImagesCount: &zero, Extensions: map[string]interface{}{"large": int64(9007199254740993)}})
	if err != nil || !strings.Contains(raw, `"historyMessages":[]`) || !strings.Contains(raw, `"imagesCount":0`) || !strings.Contains(raw, `9007199254740993`) {
		t.Fatal(raw, err)
	}
	raw, err = agentSerialize(AgentEventLLMOutput, AgentEventLLMOutputData{AssistantTexts: []string{}, Thinking: false, Usage: AgentEventLLMUsage{CacheWrite: &zero}})
	if err != nil || !strings.Contains(raw, `"assistantTexts":[]`) || !strings.Contains(raw, `"thinking":false`) || !strings.Contains(raw, `"cacheWrite":0`) {
		t.Fatal(raw, err)
	}
	for _, data := range []interface{}{AgentEventLLMInputData{Model: "m", Prompt: "p", Extensions: map[string]interface{}{"sessionId": "conflict"}}, AgentEventLLMOutputData{AssistantTexts: []string{}, Usage: AgentEventLLMUsage{Extensions: map[string]interface{}{"input": 1}}}} {
		if _, err := agentSerialize(AgentEventLLMInput, data); err == nil {
			t.Fatal("扩展覆盖未拒绝")
		}
	}
	nonfinite := math.Inf(1)
	if _, err := agentSerialize(AgentEventLLMOutput, AgentEventLLMOutputData{AssistantTexts: []string{}, Usage: AgentEventLLMUsage{CacheWrite: &nonfinite}}); err == nil {
		t.Fatal("cacheWrite 非有限值未拒绝")
	}
	logger := agentLogger()
	NewClient("", "secret", &ClientOptions{Logger: logger}).Agent.Report(context.Background(), agentEvent())
	agentWait(t, logger.failures)
}
func TestAgent进行中取消与网络错误安全(t *testing.T) {
	started := make(chan map[string]interface{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		started <- nil
		if r.URL.Query().Get("appid") == "slow" {
			<-r.Context().Done()
			return
		}
		fmt.Fprint(w, `{"errcode":0}`)
	}))
	logger := agentLogger()
	client := NewClient("slow", "SENSITIVE", &ClientOptions{APIBaseURL: server.URL, Logger: logger})
	ctx, cancel := context.WithCancel(context.Background())
	client.Agent.Report(ctx, agentEvent())
	agentWait(t, started)
	cancel()
	failure := agentWait(t, logger.failures)
	if strings.Contains(fmt.Sprint(failure), "SENSITIVE") || strings.Contains(fmt.Sprint(failure), "secret=") {
		t.Fatal("泄漏鉴权")
	}
	// 后续独立 context 仍可发起请求；改为短超时以结束测试服务器上的等待。
	next, nextCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer nextCancel()
	client.Agent.Report(next, agentEvent())
	agentWait(t, started)
	agentWait(t, logger.failures)
	server.Close()
	client.Agent.Report(context.Background(), agentEvent())
	failure = agentWait(t, logger.failures)
	if strings.Contains(fmt.Sprint(failure), "SENSITIVE") {
		t.Fatal("网络错误泄漏鉴权")
	}
}
func TestAgent成功日志恐慌不阻塞队列(t *testing.T) {
	received := make(chan map[string]interface{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received <- nil; fmt.Fprint(w, `{"errcode":0}`) }))
	defer server.Close()
	logger := agentLogger()
	logger.panicResult = true
	client := NewClient("app", "secret", &ClientOptions{APIBaseURL: server.URL, Logger: logger})
	client.Agent.Report(context.Background(), agentEvent())
	client.Agent.Report(context.Background(), agentEvent())
	agentWait(t, received)
	agentWait(t, received)
}

func TestAgent工具大整数不丢精度(t *testing.T) {
	raw, err := agentSerialize(AgentEventAfterToolCall, AgentEventAfterToolCallData{ToolName: "tool", Result: int64(9007199254740993)})
	if err != nil || !strings.Contains(raw, `"result":9007199254740993`) {
		t.Fatal(raw, err)
	}
}
