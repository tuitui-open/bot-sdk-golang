package tuitui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/websocket"
	"strings"
	"testing"
)

func Test事件可读取机器人名称(t *testing.T) {
	if (EventBody{}).BotName() != "" {
		t.Fatal("缺少机器人名称时应返回空字符串")
	}
	body := EventBody{"bot_name": "测试机器人"}
	if body.BotName() != "测试机器人" {
		t.Fatalf("机器人名称不正确：%q", body.BotName())
	}
}

func Test事件可读取发送者部门路径(t *testing.T) {
	if (EventBody{}).UserDeptPath() != "" {
		t.Fatal("缺少部门路径时应返回空字符串")
	}
	body := EventBody{"user_dept_path": "企业-研发中心-平台部"}
	if body.UserDeptPath() != "企业-研发中心-平台部" {
		t.Fatalf("发送者部门路径不正确：%q", body.UserDeptPath())
	}
}

func Test正文渲染名片聊天正文和文件(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"msg_type": "card",
		"text":     "聊天正文",
		"card": map[string]interface{}{
			"name":    "张三",
			"account": "zhangsan",
		},
		"images": []interface{}{
			"https://example.com/plain.png",
			"https://example.com/diagram.png",
		},
		"file": map[string]interface{}{"name": "说明.txt", "url": "https://example.com/readme.txt"},
	}
	want := strings.Join([]string{
		"聊天正文",
		"[图片] https://example.com/plain.png",
		"[图片] https://example.com/diagram.png",
		"[文件] 说明.txt: https://example.com/readme.txt",
		"[名片]",
		"姓名: 张三",
		"推推账号: zhangsan",
	}, "\n")
	if got := RenderMessageBody(data); got != want {
		t.Fatalf("正文渲染错误：\n%s", got)
	}
}

func Test合并转发递归渲染并提取详细媒体(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"msg_type": "merged",
		"merged": map[string]interface{}{
			"source": "项目群",
			"msgs": []interface{}{
				map[string]interface{}{
					"user_name":    "张三",
					"user_account": "zhangsan",
					"timestamp":    float64(1723420800000),
					"msg_type":     "file",
					"file": map[string]interface{}{
						"name":    "报告.pdf",
						"url":     "https://example.com/report.pdf",
						"file_id": "file-1",
					},
				},
				map[string]interface{}{
					"user_name": "李四",
					"msg_type":  "merged",
					"merged": map[string]interface{}{
						"source": "子群",
						"msgs": []interface{}{
							map[string]interface{}{
								"user_name": "王五",
								"msg_type":  "image",
								"images":    []interface{}{"https://example.com/screenshot.png"},
							},
						},
					},
				},
			},
		},
	}

	rendered := RenderMessageBody(data)
	for _, expected := range []string{
		"[合并转发：项目群]",
		"时间: ",
		"发言人: 张三 (zhangsan)",
		"[文件] 报告.pdf: https://example.com/report.pdf",
		"[合并转发：子群]",
		"发言人: 王五",
		"[图片] https://example.com/screenshot.png",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("合并转发缺少 %q：\n%s", expected, rendered)
		}
	}
	media := GetDetailedMessageMedia(data)
	if len(media) != 2 {
		t.Fatalf("媒体数量错误：%#v", media)
	}
	if media[0].Name != "报告.pdf" || media[0].FileID != "file-1" || media[0].URL != "https://example.com/report.pdf" {
		t.Fatalf("文件信息错误：%#v", media[0])
	}
	if media[1].Name != "" || media[1].MIMEType != "image/*" {
		t.Fatalf("图片信息错误：%#v", media[1])
	}
}

func Test媒体提取支持单文件和引用(t *testing.T) {
	t.Parallel()
	file := map[string]interface{}{
		"name": "第一份.txt",
		"url":  "https://example.com/first.txt",
		"fid":  "fid-1",
	}
	data := map[string]interface{}{
		"file": file,
		"ref": map[string]interface{}{
			"user_name":    "引用者",
			"user_account": "reference",
			"images":       []interface{}{"https://example.com/reference.png"},
		},
	}
	media := GetDetailedMessageMedia(data)
	if len(media) != 2 {
		t.Fatalf("媒体数量错误：%#v", media)
	}
	if media[0].FileID != "fid-1" || media[1].MIMEType != "image/*" {
		t.Fatalf("媒体元数据错误：%#v", media)
	}
	legacy := GetMessageMedia(data)
	if len(legacy) != 2 || legacy[0].URL != media[0].URL {
		t.Fatalf("旧媒体接口不兼容：%#v", legacy)
	}
	rendered := RenderMessageBody(data)
	if !strings.Contains(rendered, "[引用来自 引用者 (reference) 的消息，内容如下]") || !strings.Contains(rendered, "[图片] https://example.com/reference.png") {
		t.Fatalf("引用消息渲染错误：\n%s", rendered)
	}
	if !strings.Contains(rendered, "[文件] 第一份.txt: https://example.com/first.txt") {
		t.Fatalf("重复文件没有保留完整名称：\n%s", rendered)
	}
}

func Test设备类型常量与分类方法一致(t *testing.T) {
	constants := []string{DeviceTypeUnknown, DeviceTypeWindows, DeviceTypeMac,
		DeviceTypeAndroid, DeviceTypeIOS, DeviceTypeWeb, DeviceTypeServer,
		DeviceTypeCLI, DeviceTypePad, DeviceTypeIPad}
	for index, value := range constants {
		if value != fmt.Sprint(index) {
			t.Fatalf("设备常量编码错误：%d=%s", index, value)
		}
	}
	client := NewClient("app", "secret", nil)
	cases := []struct {
		value      string
		pc, mobile bool
	}{
		{"0", false, false}, {"1", true, false}, {"2", true, false},
		{"3", false, true}, {"4", false, true}, {"5", true, false},
		{"6", false, false}, {"7", false, false}, {"8", false, true},
		{"9", false, true}, {"99", false, false}, {"", false, false},
	}
	for _, item := range cases {
		if IsPCDevice(item.value) != item.pc || client.Event.IsPCDevice(item.value) != item.pc {
			t.Fatalf("设备 %q 的 PC 分类错误", item.value)
		}
		if IsMobileDevice(item.value) != item.mobile || client.Event.IsMobileDevice(item.value) != item.mobile {
			t.Fatalf("设备 %q 的移动端分类错误", item.value)
		}
	}
	if (EventBody{}).DeviceType() != DeviceTypeUnknown {
		t.Fatal("缺少设备字段时应返回未知设备类型")
	}
}

func Test事件透传设备类型并在缺失时返回未知类型(t *testing.T) {
	devices := []interface{}{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "99", "", nil}
	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		defer socket.Close()
		for index, device := range devices {
			envelope := map[string]interface{}{
				"event_id": fmt.Sprintf("device-%d", index),
				"body": map[string]interface{}{
					"event": "future_event", "future_body": true,
					"data": map[string]interface{}{"future_data": "保留"},
				},
			}
			if device != nil {
				envelope["header"] = map[string]interface{}{"X-Tuitui-Robot-From-Device-Type": device}
			}
			if err := socket.WriteJSON(envelope); err != nil {
				serverErrors <- err
				return
			}
			var ack map[string]interface{}
			if err := socket.ReadJSON(&ack); err != nil {
				serverErrors <- err
				return
			}
			if ack["ack"] != envelope["event_id"] {
				serverErrors <- fmt.Errorf("ACK 错误：%v", ack)
				return
			}
		}
		serverErrors <- nil
		socket.ReadMessage()
	}))
	defer server.Close()
	client := NewClient("app", "secret", &ClientOptions{WebSocketBaseURL: "ws" + strings.TrimPrefix(server.URL, "http")})
	events := make(chan EventBody, len(devices))
	subscription := client.Event.Subscribe(context.Background(), &SubscribeOptions{
		OnEvent: func(body EventBody) { events <- body },
	})
	defer subscription.Unsubscribe()
	for _, device := range devices {
		want, ok := device.(string)
		if !ok {
			want = DeviceTypeUnknown
		}
		select {
		case body := <-events:
			if body.DeviceType() != want || body["device_type"] != want {
				t.Fatalf("设备类型错误：期望 %q，收到 %#v", want, body)
			}
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded["device_type"] != want || decoded["future_body"] != true {
				t.Fatalf("序列化后的事件字段错误：%s", encoded)
			}
			if body["data"].(map[string]interface{})["future_data"] != "保留" {
				t.Fatal("正文未知字段应保留")
			}
			if _, ok := body["header"]; ok {
				t.Fatal("回调不应暴露完整 header")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("等待事件超时")
		}
	}
	if err := <-serverErrors; err != nil {
		t.Fatal(err)
	}
}
