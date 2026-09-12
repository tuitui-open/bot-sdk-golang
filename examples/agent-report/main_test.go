package main

import (
	"context"
	"encoding/json"
	"fmt"
	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func Test消息触发与过滤(t *testing.T) {
	for _, event := range []string{tuitui.EventSingleChat, tuitui.EventGroupChat, tuitui.EventTeamsPostCreate, "unknown"} {
		for _, at := range []bool{true, false} {
			body := tuitui.EventBody{"event": event, "data": map[string]interface{}{"at_me": at}}
			want := event == tuitui.EventSingleChat || ((event == tuitui.EventGroupChat || event == tuitui.EventTeamsPostCreate) && at)
			if shouldReport(body) != want {
				t.Fatal(event, at)
			}
		}
	}
}
func Test完整流程并发与取消(t *testing.T) {
	received := make(chan string, 24)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p []map[string]interface{}
		json.NewDecoder(r.Body).Decode(&p)
		received <- p[0]["event"].(string)
		fmt.Fprint(w, `{"errcode":0}`)
	}))
	defer server.Close()
	client := tuitui.NewClient("app", "secret", &tuitui.ClientOptions{APIBaseURL: server.URL})
	body := tuitui.EventBody{"event": tuitui.EventSingleChat, "user_account": "a", "data": map[string]interface{}{"msgid": "m"}}
	var waits []time.Duration
	err := reportFlow(context.Background(), client, body, func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(waits) != 2 || waits[0] != 2*time.Second || waits[1] != time.Second {
		t.Fatal(waits)
	}
	names := []string{"message_received", "llm_input", "llm_output", "before_tool_call", "after_tool_call", "subagent_spawned", "llm_input", "llm_output", "before_tool_call", "after_tool_call", "subagent_ended", "agent_end"}
	for _, name := range names {
		select {
		case actual := <-received:
			if actual != name {
				t.Fatal(actual, name)
			}
		case <-time.After(time.Second):
			t.Fatal("事件缺失")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	var tasks sync.WaitGroup
	entered := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		tasks.Add(1)
		go func() {
			defer tasks.Done()
			err := reportFlow(ctx, client, body, func(ctx context.Context, _ time.Duration) error { entered <- struct{}{}; return wait(ctx, time.Hour) })
			if err != context.Canceled {
				t.Error(err)
			}
		}()
	}
	<-entered
	<-entered
	cancel()
	tasks.Wait()
}
