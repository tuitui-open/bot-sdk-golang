package sample

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func ParseTarget(arguments []string) (tuitui.ToTarget, error) {
	flags := flag.NewFlagSet("target", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	account := flags.String("account", "", "接收者账号")
	uid := flags.String("uid", "", "接收者 UID")
	group := flags.String("group", "", "接收群 ID")
	if err := flags.Parse(arguments); err != nil {
		return tuitui.ToTarget{}, targetUsage(err)
	}
	if flags.NArg() != 0 {
		return tuitui.ToTarget{}, targetUsage(fmt.Errorf("不支持位置参数"))
	}

	values := []string{strings.TrimSpace(*account), strings.TrimSpace(*uid), strings.TrimSpace(*group)}
	selected := 0
	for _, value := range values {
		if value != "" {
			selected++
		}
	}
	if selected != 1 {
		return tuitui.ToTarget{}, targetUsage(fmt.Errorf("发送目标必须且只能指定一个"))
	}

	to := tuitui.ToAPI{}
	switch {
	case values[0] != "":
		return to.Account(values[0]), nil
	case values[1] != "":
		return to.UID(values[1]), nil
	default:
		return to.Group(values[2]), nil
	}
}

func ParsePostTarget(arguments []string) (string, string, error) {
	flags := flag.NewFlagSet("post-target", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	team := flags.String("team", "", "团队 ID")
	channel := flags.String("channel", "", "频道 ID")
	if err := flags.Parse(arguments); err != nil {
		return "", "", postUsage(err)
	}
	if flags.NArg() != 0 {
		return "", "", postUsage(fmt.Errorf("不支持位置参数"))
	}

	teamID := strings.TrimSpace(*team)
	channelID := strings.TrimSpace(*channel)
	if teamID == "" || channelID == "" {
		return "", "", postUsage(fmt.Errorf("团队 ID 和频道 ID 都不能为空"))
	}
	return teamID, channelID, nil
}

func PrintResponse(message string, response tuitui.APIResponse) {
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Printf("%s\n%v\n", message, response)
		return
	}
	fmt.Printf("%s\n%s\n", message, data)
}

func targetUsage(err error) error {
	return fmt.Errorf("%w\n用法: --account <账号> | --uid <UID> | --group <群ID>", err)
}

func postUsage(err error) error {
	return fmt.Errorf("%w\n用法: --team <团队ID> --channel <频道ID>", err)
}
