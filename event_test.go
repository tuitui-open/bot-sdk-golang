package tuitui

import (
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
