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
	sub, err := client.Agent.BuildSubagentContext(tuitui.BuildAgentSubagentContextOptions{RequesterContext: base})
	if err != nil {
		t.Fatal(err)
	}
	tool := tuitui.AgentEventToolContext{Context: base, ToolCallID: "parent-tool"}
	childTool := tuitui.AgentEventToolContext{Context: sub, ToolCallID: "child-tool"}
	events := []tuitui.AgentEvent{
		tuitui.AgentMessageReceivedEvent{
			Context: base,
			Data: tuitui.AgentEventMessageReceivedData{
				Content: "验收",
			},
		},
		tuitui.AgentLLMInputEvent{
			Context: base,
			Data: tuitui.AgentEventLLMInputData{
				Model:  "test",
				Prompt: "验收",
			},
		},
		tuitui.AgentLLMOutputEvent{
			Context: base,
			Data: tuitui.AgentEventLLMOutputData{
				AssistantTexts: []string{"开始查询"},
				Thinking:       "查询资料",
			},
		},
		tuitui.AgentBeforeToolCallEvent{
			Context: tool,
			Data: tuitui.AgentEventBeforeToolCallData{
				ToolName: "lookup",
			},
		},
		tuitui.AgentAfterToolCallEvent{
			Context: tool,
			Data: tuitui.AgentEventAfterToolCallData{
				ToolName: "lookup",
				Result:   map[string]interface{}{"found": true},
			},
		},
		tuitui.AgentSubagentSpawnedEvent{Context: sub},
		tuitui.AgentLLMInputEvent{
			Context: sub,
			Data: tuitui.AgentEventLLMInputData{
				Model:  "test",
				Prompt: "总结",
			},
		},
		tuitui.AgentLLMOutputEvent{
			Context: sub,
			Data: tuitui.AgentEventLLMOutputData{
				AssistantTexts: []string{"完成"},
				Thinking:       nil,
			},
		},
		tuitui.AgentBeforeToolCallEvent{
			Context: childTool,
			Data: tuitui.AgentEventBeforeToolCallData{
				ToolName: "summarize",
			},
		},
		tuitui.AgentAfterToolCallEvent{
			Context: childTool,
			Data: tuitui.AgentEventAfterToolCallData{
				ToolName: "summarize",
				Result:   "完成",
			},
		},
		tuitui.AgentSubagentEndedEvent{
			Context: sub,
			Data: tuitui.AgentEventSubagentEndedData{
				Outcome: tuitui.AgentSubagentOK,
			},
		},
		tuitui.AgentEndEvent{
			Context: base,
			Data: tuitui.AgentEventEndData{
				Success: true,
				Status:  tuitui.AgentEndDone,
			},
		},
	}
	for _, event := range events {
		client.Agent.Report(ctx, event)
	}
	for range events {
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
