package tuitui

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func agentBase() AgentEventContext {
	return AgentEventContext{SessionKey: "agent:app:tuitui:group:g", MessageID: "m", RunID: "r"}
}

func agentServer(t *testing.T, expected int) (*httptest.Server, <-chan []map[string]interface{}) {
	t.Helper()
	var mu sync.Mutex
	requests := make([]map[string]interface{}, 0, expected)
	done := make(chan []map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		var payload []map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		requests = append(requests, payload[0])
		if len(requests) == expected {
			done <- append([]map[string]interface{}{}, requests...)
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	return server, done
}

func waitAgentRequests(t *testing.T, done <-chan []map[string]interface{}) []map[string]interface{} {
	t.Helper()
	select {
	case requests := <-done:
		return requests
	case <-time.After(5 * time.Second):
		t.Fatal("等待 Agent 上报超时")
	}
	return nil
}

func testAgentClient(url string) *Client {
	return NewClient("app", "secret", &ClientOptions{APIBaseURL: url})
}

func TestAgent上下文由结构化目标和入站消息生成(t *testing.T) {
	client := NewClient("app", "secret", nil)
	ctx, err := client.Agent.BuildContext(BuildAgentContextOptions{
		Target: AgentContextTarget{Type: "direct", Account: "alice"}, MessageID: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.SessionKey != "agent:app:tuitui:direct:alice" || ctx.RunID == "" {
		t.Fatalf("上下文错误: %#v", ctx)
	}
	if _, err = client.Agent.BuildContext(BuildAgentContextOptions{Target: AgentContextTarget{Type: "group", GroupID: "bad:id"}, MessageID: "m"}); err == nil {
		t.Fatal("非法会话组成字段应失败")
	}
}

func TestAgent具体方法生成八类事件并保序(t *testing.T) {
	server, done := agentServer(t, 8)
	defer server.Close()
	client := testAgentClient(server.URL)
	ctx := agentBase()
	sub, err := client.Agent.BuildSubagentContext(BuildAgentSubagentContextOptions{ParentContext: ctx})
	if err != nil {
		t.Fatal(err)
	}
	client.Agent.Report.MessageReceived(context.Background(), ctx)
	client.Agent.Report.LLMInput(context.Background(), ctx, AgentEventLLMInputData{Model: "model", Prompt: "查询北京天气"})
	client.Agent.Report.LLMOutput(context.Background(), ctx, AgentEventLLMOutputData{AssistantTexts: []string{}, Thinking: "调用工具", Usage: AgentEventLLMUsage{Input: 10, Output: 5}})
	client.Agent.Report.BeforeToolCall(context.Background(), ctx, AgentEventBeforeToolCallData{ToolCallID: "call", ToolName: "weather"})
	client.Agent.Report.AfterToolCall(context.Background(), ctx, AgentEventAfterToolCallData{ToolCallID: "call", ToolName: "weather", Result: "晴"})
	client.Agent.Report.SubagentSpawned(context.Background(), sub, AgentEventSubagentSpawnedData{})
	client.Agent.Report.SubagentEnded(context.Background(), sub, AgentEventSubagentEndedData{Outcome: AgentSubagentOK})
	client.Agent.Report.AgentEnd(context.Background(), ctx)
	requests := waitAgentRequests(t, done)
	names := []string{"message_received", "llm_input", "llm_output", "before_tool_call", "after_tool_call", "subagent_spawned", "subagent_ended", "agent_end"}
	for i, request := range requests {
		if request["event"] != names[i] {
			t.Fatalf("第 %d 条事件为 %v", i, request["event"])
		}
		if i > 0 && request["timestamp"].(float64) <= requests[i-1]["timestamp"].(float64) {
			t.Fatal("时间戳未递增")
		}
	}
	var first, output, tool, last map[string]interface{}
	_ = json.Unmarshal([]byte(requests[0]["data"].(string)), &first)
	_ = json.Unmarshal([]byte(requests[2]["data"].(string)), &output)
	_ = json.Unmarshal([]byte(requests[3]["data"].(string)), &tool)
	_ = json.Unmarshal([]byte(requests[7]["data"].(string)), &last)
	if len(first) != 0 || tool["toolCallId"] != "call" || len(last) != 0 {
		t.Fatal("事件 data 不符合协议")
	}
	usage := output["usage"].(map[string]interface{})
	if usage["cacheRead"] != float64(0) || usage["total"] != float64(15) {
		t.Fatalf("usage 默认值错误: %#v", usage)
	}
}

func TestAgent超过一百条仍按客户端顺序发送(t *testing.T) {
	server, done := agentServer(t, 150)
	defer server.Close()
	client := testAgentClient(server.URL)
	for i := 0; i < 150; i++ {
		client.Agent.Report.LLMInput(context.Background(), agentBase(), AgentEventLLMInputData{
			Model: "model", Prompt: strconv.Itoa(i), Extensions: map[string]interface{}{"index": i},
		})
	}
	requests := waitAgentRequests(t, done)
	for i, request := range requests {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(request["data"].(string)), &data); err != nil {
			t.Fatal(err)
		}
		if int(data["index"].(float64)) != i {
			t.Fatalf("上报顺序错误: %v", data["index"])
		}
	}
}

func TestAgent上下文字段按实际值原样上报(t *testing.T) {
	server, done := agentServer(t, 2)
	defer server.Close()
	client := testAgentClient(server.URL)
	client.Agent.Report.MessageReceived(context.Background(), AgentEventContext{
		SessionKey: "agent:app:tuitui:group:g", MessageID: "message",
	})
	client.Agent.Report.LLMInput(context.Background(), AgentEventContext{
		SessionKey: "agent:app:tuitui:group:g", RunID: "run",
	}, AgentEventLLMInputData{Model: "model", Prompt: "prompt"})
	requests := waitAgentRequests(t, done)
	messageContext := requests[0]["ctx"].(map[string]interface{})
	runContext := requests[1]["ctx"].(map[string]interface{})
	if messageContext["messageId"] != "message" || messageContext["runId"] != nil {
		t.Fatalf("message_received ctx 错误: %#v", messageContext)
	}
	if runContext["runId"] != "run" || runContext["messageId"] != nil {
		t.Fatalf("llm_input ctx 错误: %#v", runContext)
	}
}

func TestAgent非法参数和请求失败不向调用方抛出(t *testing.T) {
	client := NewClient("", "secret", nil)
	client.Agent.Report.MessageReceived(nil, agentBase())
	client.Agent.Report.LLMInput(context.Background(), agentBase(), AgentEventLLMInputData{})
}
