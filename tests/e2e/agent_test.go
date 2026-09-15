//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

type agentLog struct {
	success chan map[string]interface{}
	failed  chan string
}

func (l *agentLog) Debug(string, ...interface{}) {}
func (l *agentLog) Warn(string, ...interface{})  {}
func (l *agentLog) Info(_ string, v ...interface{}) {
	l.success <- v[0].(map[string]interface{})
}
func (l *agentLog) Error(_ string, v ...interface{}) {
	l.failed <- fmt.Sprint(v...)
}

func TestAgent群消息上报十二条事件(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_GROUP")
	logger := &agentLog{
		success: make(chan map[string]interface{}, 12),
		failed:  make(chan string, 12),
	}
	client := tuitui.NewClient(
		os.Getenv("TUITUI_BOT_APPID"),
		os.Getenv("TUITUI_BOT_SECRET"),
		&tuitui.ClientOptions{
			APIBaseURL: os.Getenv("TUITUI_BOT_API_BASE_URL"),
			Logger:     logger,
		},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := client.IM.SendText(ctx, tuitui.SendIMTextOptions{
		To:   client.To.Group(os.Getenv("TARGET_GROUP")),
		Text: "Go Agent 上报验收",
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := client.Agent.BuildContext(tuitui.BuildAgentContextOptions{
		Target: tuitui.AgentContextTarget{
			Type:    "group",
			GroupID: os.Getenv("TARGET_GROUP"),
		},
		MessageID: requireResponseID(t, response, "msgid", "message_id"),
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := client.Agent.BuildSubagentContext(tuitui.BuildAgentSubagentContextOptions{ParentContext: base})
	if err != nil {
		t.Fatal(err)
	}
	cacheWrite := float64(0)
	client.Agent.Report.MessageReceived(ctx, base)
	client.Agent.Report.LLMInput(ctx, base, tuitui.AgentEventLLMInputData{Model: "test", Prompt: "验收"})
	client.Agent.Report.LLMOutput(ctx, base, tuitui.AgentEventLLMOutputData{AssistantTexts: []string{}, Thinking: "查询资料", Usage: tuitui.AgentEventLLMUsage{Input: 1, Output: 1, CacheRead: 0, CacheWrite: &cacheWrite, Total: 2}})
	client.Agent.Report.BeforeToolCall(ctx, base, tuitui.AgentEventBeforeToolCallData{ToolCallID: "parent-tool", ToolName: "lookup"})
	client.Agent.Report.AfterToolCall(ctx, base, tuitui.AgentEventAfterToolCallData{ToolCallID: "parent-tool", ToolName: "lookup", Result: map[string]interface{}{"found": true}})
	client.Agent.Report.SubagentSpawned(ctx, sub, tuitui.AgentEventSubagentSpawnedData{})
	client.Agent.Report.LLMInput(ctx, sub, tuitui.AgentEventLLMInputData{Model: "test", Prompt: "总结"})
	client.Agent.Report.LLMOutput(ctx, sub, tuitui.AgentEventLLMOutputData{AssistantTexts: []string{"完成"}, Thinking: "", Usage: tuitui.AgentEventLLMUsage{Input: 1, Output: 1, CacheRead: 0, CacheWrite: &cacheWrite, Total: 2}})
	client.Agent.Report.BeforeToolCall(ctx, sub, tuitui.AgentEventBeforeToolCallData{ToolCallID: "child-tool", ToolName: "summarize"})
	client.Agent.Report.AfterToolCall(ctx, sub, tuitui.AgentEventAfterToolCallData{ToolCallID: "child-tool", ToolName: "summarize", Result: "完成"})
	client.Agent.Report.SubagentEnded(ctx, sub, tuitui.AgentEventSubagentEndedData{Outcome: tuitui.AgentSubagentOK})
	client.Agent.Report.AgentEnd(ctx, base)
	for index := 0; index < 12; index++ {
		select {
		case result := <-logger.success:
			r := result["response"].(tuitui.APIResponse)
			id, ok := r["trans_id"].(string)
			if r["errcode"] != float64(0) || !ok || strings.TrimSpace(id) == "" {
				t.Fatalf("无效响应: %v", r)
			}
		case failure := <-logger.failed:
			t.Fatal(failure)
		case <-ctx.Done():
			t.Fatal("未收到全部 12 条响应")
		}
	}
	select {
	case failure := <-logger.failed:
		t.Fatal(failure)
	default:
	}
}
