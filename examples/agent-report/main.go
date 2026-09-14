package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
	"github.com/tuitui-open/bot-sdk-golang/examples/internal/sample"
)

type logger struct{}

func (logger) Debug(string, ...interface{})                {}
func (logger) Info(message string, values ...interface{})  { log.Println(message, values) }
func (logger) Warn(message string, values ...interface{})  { log.Println(message, values) }
func (logger) Error(message string, values ...interface{}) { log.Println(message, values) }
func main() {
	app, secret, options, err := sample.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	options.Logger = logger{}
	client := tuitui.NewClient(app, secret, options)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bot, err := client.Property.Info(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("正在监听 %s 的 Agent 请求", bot.Name)
	var tasks sync.WaitGroup
	subscription := client.Event.Subscribe(ctx, &tuitui.SubscribeOptions{OnEvent: func(body tuitui.EventBody) {
		if !shouldReport(body) {
			return
		}
		tasks.Add(1)
		go func() {
			defer tasks.Done()
			if err := reportFlow(ctx, client, body, wait); err != nil {
				log.Printf("示例执行失败: %v", err)
			}
		}()
	}, OnError: func(err error) { log.Printf("订阅失败: %v", err) }})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
	signal.Stop(signals)
	cancel()
	subscription.Unsubscribe()
	tasks.Wait()
}
func shouldReport(body tuitui.EventBody) bool {
	data, _ := body["data"].(map[string]interface{})
	return body["event"] == tuitui.EventSingleChat || ((body["event"] == tuitui.EventGroupChat || body["event"] == tuitui.EventTeamsPostCreate) && data["at_me"] == true)
}
func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func reportFlow(ctx context.Context, client *tuitui.Client, body tuitui.EventBody, pause func(context.Context, time.Duration) error) error {
	started := time.Now()
	base, err := client.Agent.ContextFromMessage(body, nil)
	if err != nil {
		return err
	}
	data, _ := body["data"].(map[string]interface{})
	content := client.Event.RenderMessageBody(data)
	if body["event"] == tuitui.EventTeamsPostCreate {
		content, _ = data["content"].(string)
	}
	if content == "" {
		content = "收到消息"
	}
	// base.SessionKey 路由会话，MessageID 关联入站消息，RunID 标识本次执行。
	// Content 是收到的正文；Model/Prompt 是模型及输入。
	sub, err := client.Agent.BuildSubagentContext(tuitui.BuildAgentSubagentContextOptions{RequesterContext: base})
	if err != nil {
		return err
	}
	client.Agent.Report(ctx, tuitui.AgentMessageReceivedEvent{Context: base, Data: tuitui.AgentEventMessageReceivedData{Content: content}})
	client.Agent.Report(ctx, tuitui.AgentLLMInputEvent{Context: base, Data: tuitui.AgentEventLLMInputData{Model: "sample-model", Prompt: content}})
	if err := pause(ctx, 2*time.Second); err != nil {
		return err
	}
	cacheWrite := float64(0)
	// AssistantTexts/Thinking 是模型回复与思考；Usage 记录输入、输出、缓存读取、
	// 缓存写入和总 token 数。
	client.Agent.Report(ctx, tuitui.AgentLLMOutputEvent{Context: base, Data: tuitui.AgentEventLLMOutputData{AssistantTexts: []string{"查询资料"}, Thinking: "先查询再总结", Usage: tuitui.AgentEventLLMUsage{Input: 10, Output: 5, CacheRead: 0, CacheWrite: &cacheWrite, Total: 15}}})
	// ToolCallID 关联工具开始/结束；ToolName、Params、Result 是工具名、参数和结果。
	tool := tuitui.AgentEventToolContext{Context: base, ToolCallID: "lookup"}
	client.Agent.Report(ctx, tuitui.AgentBeforeToolCallEvent{Context: tool, Data: tuitui.AgentEventBeforeToolCallData{ToolName: "lookup", Params: map[string]interface{}{"input": content}}})
	if err := pause(ctx, time.Second); err != nil {
		return err
	}
	client.Agent.Report(ctx, tuitui.AgentAfterToolCallEvent{Context: tool, Data: tuitui.AgentEventAfterToolCallData{ToolName: "lookup", Result: map[string]interface{}{"found": true}}})
	// 子 Agent Context 的 RequesterSessionKey 指回父会话；AgentID/Label 描述子 Agent，
	// Outcome/Reason 记录其结束结果及可选原因。
	client.Agent.Report(ctx, tuitui.AgentSubagentSpawnedEvent{Context: sub, Data: tuitui.AgentEventSubagentSpawnedData{AgentID: "Explore", Label: "总结助手"}})
	client.Agent.Report(ctx, tuitui.AgentLLMInputEvent{Context: sub, Data: tuitui.AgentEventLLMInputData{Model: "sample-model", Prompt: "总结资料"}})
	client.Agent.Report(ctx, tuitui.AgentLLMOutputEvent{Context: sub, Data: tuitui.AgentEventLLMOutputData{AssistantTexts: []string{"完成总结"}, Thinking: "", Usage: tuitui.AgentEventLLMUsage{Input: 6, Output: 3, CacheRead: 0, CacheWrite: &cacheWrite, Total: 9}}})
	childTool := tuitui.AgentEventToolContext{Context: sub, ToolCallID: "summarize"}
	client.Agent.Report(ctx, tuitui.AgentBeforeToolCallEvent{Context: childTool, Data: tuitui.AgentEventBeforeToolCallData{ToolName: "summarize"}})
	client.Agent.Report(ctx, tuitui.AgentAfterToolCallEvent{Context: childTool, Data: tuitui.AgentEventAfterToolCallData{ToolName: "summarize", Result: "完成"}})
	client.Agent.Report(ctx, tuitui.AgentSubagentEndedEvent{Context: sub, Data: tuitui.AgentEventSubagentEndedData{Outcome: tuitui.AgentSubagentOK, Reason: "completed"}})
	duration := float64(time.Since(started)) / float64(time.Millisecond)
	// AgentEnd Data 接受任意 JSON 对象；这里记录是否成功和实际耗时。
	client.Agent.Report(ctx, tuitui.AgentEndEvent{Context: base, Data: map[string]interface{}{"success": true, "durationMs": duration}})
	return nil
}
