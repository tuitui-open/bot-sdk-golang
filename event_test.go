package tuitui

import "testing"

func Test事件可读取机器人名称(t *testing.T) {
	if (EventBody{}).BotName() != "" {
		t.Fatal("缺少机器人名称时应返回空字符串")
	}
	body := EventBody{"bot_name": "测试机器人"}
	if body.BotName() != "测试机器人" {
		t.Fatalf("机器人名称不正确：%q", body.BotName())
	}
}
