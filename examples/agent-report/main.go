package main

import (
	"context"
	"log"
	"os"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/internal/dotenv"
)

func main() {
	if err := dotenv.LoadClosest(); err != nil {
		log.Fatal(err)
	}
	client := tuitui.NewClient(
		os.Getenv("TUITUI_BOT_APPID"),
		os.Getenv("TUITUI_BOT_SECRET"),
		nil,
	)
	reportSubagent := false
	for _, arg := range os.Args[1:] {
		if arg == "--subagent" {
			reportSubagent = true
		}
	}
	ctx := context.Background()
	client.Event.Subscribe(ctx, &tuitui.SubscribeOptions{OnConnected: func() {
		bot, err := client.Property.Info(ctx)
		if err != nil {
			log.Printf("查询机器人信息失败: %v", err)
			return
		}
		log.Printf("机器人 `%s` 等待收消息。请私聊此机器人，或在群聊、团队帖子中 @该机器人", bot.Name)
	}, OnEvent: func(body tuitui.EventBody) {
		if !shouldReport(body) {
			return
		}
		go reportFlow(ctx, client, body, reportSubagent)
	}, OnError: func(err error) { log.Print(err) }})
	select {}
}
func shouldReport(body tuitui.EventBody) bool {
	data, _ := body["data"].(map[string]interface{})
	return body["event"] == tuitui.EventSingleChat || ((body["event"] == tuitui.EventGroupChat || body["event"] == tuitui.EventTeamsPostCreate) && data["at_me"] == true)
}
func reportFlow(ctx context.Context, client *tuitui.Client, body tuitui.EventBody, reportSubagent bool) error {
	base, err := client.Agent.ContextFromMessage(body)
	if err != nil {
		return err
	}
	client.Agent.Report.MessageReceived(ctx, base)
	client.Agent.Report.LLMInput(ctx, base, tuitui.AgentEventLLMInputData{Model: "sample-model", Prompt: "查询北京天气"})
	time.Sleep(2 * time.Second)
	client.Agent.Report.LLMOutput(ctx, base, tuitui.AgentEventLLMOutputData{AssistantTexts: []string{}, Thinking: "用户想要查询北京天气，我可以使用 weather 工具来查询", Usage: tuitui.AgentEventLLMUsage{Input: 10, Output: 5, CacheRead: 0, Total: 15}})
	time.Sleep(time.Second)
	// 实际使用时应传入模型输出的 toolCallId，同一次工具调用的前后事件必须成对复用。
	toolCallID := "abc1234567890"
	client.Agent.Report.BeforeToolCall(ctx, base, tuitui.AgentEventBeforeToolCallData{ToolCallID: toolCallID, ToolName: "weather", Params: map[string]interface{}{"city": "北京"}})
	time.Sleep(time.Second)
	client.Agent.Report.AfterToolCall(ctx, base, tuitui.AgentEventAfterToolCallData{ToolCallID: toolCallID, ToolName: "weather", Result: "晴，气温 26 度"})
	if reportSubagent {
		sub, err := client.Agent.BuildSubagentContext(tuitui.BuildAgentSubagentContextOptions{ParentContext: base})
		if err != nil {
			return err
		}
		client.Agent.Report.SubagentSpawned(ctx, sub, tuitui.AgentEventSubagentSpawnedData{AgentID: "Explore", Label: "补充检查工具结果"})
		client.Agent.Report.LLMInput(ctx, sub, tuitui.AgentEventLLMInputData{Model: "sample-model", Prompt: "复核主 Agent 的工具结果"})
		client.Agent.Report.LLMOutput(ctx, sub, tuitui.AgentEventLLMOutputData{AssistantTexts: []string{"子 Agent 复核完成"}, Thinking: "确认工具结果符合预期", Usage: tuitui.AgentEventLLMUsage{Input: 6, Output: 3, CacheRead: 0, Total: 9}})
		childToolCallID := "def1234567890"
		client.Agent.Report.BeforeToolCall(ctx, sub, tuitui.AgentEventBeforeToolCallData{ToolCallID: childToolCallID, ToolName: "sample_verify", Params: map[string]interface{}{"result": "晴，气温 26 度"}})
		client.Agent.Report.AfterToolCall(ctx, sub, tuitui.AgentEventAfterToolCallData{ToolCallID: childToolCallID, ToolName: "sample_verify", Result: map[string]interface{}{"verified": true}})
		client.Agent.Report.SubagentEnded(ctx, sub, tuitui.AgentEventSubagentEndedData{Outcome: tuitui.AgentSubagentOK})
	}
	client.Agent.Report.AgentEnd(ctx, base)
	log.Print("Agent 执行中间步骤已全部加入上报队列")
	return nil
}
