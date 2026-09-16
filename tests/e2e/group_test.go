//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func TestGroup读取群信息并拉入移除和恢复群成员(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_ACCOUNT", "TARGET_GROUP")
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	groupID := os.Getenv("TARGET_GROUP")
	account := os.Getenv("TARGET_ACCOUNT")
	membersToChange := []string{account}

	groups, err := client.Group.RobotIn(ctx)
	if err != nil || !hasGroupListItem(groups, groupID) {
		t.Fatalf("机器人所在群中缺少测试群：%#v, %v", groups, err)
	}
	groups, err = client.Group.UserIsIn(ctx, tuitui.UserIsInOptions{
		User: account, Groups: []string{groupID},
	})
	if err != nil || len(groups) != 1 || groups[0].GroupID != groupID {
		t.Fatalf("用户和机器人的共同群返回不正确：%#v, %v", groups, err)
	}
	info, err := client.Group.Info(ctx, groupID)
	if err != nil {
		t.Fatalf("群信息返回不正确：%#v, %v", info, err)
	}

	restored := false
	defer func() {
		if restored {
			return
		}
		if _, restoreErr := client.Group.MemberAdd(ctx, groupID, membersToChange); restoreErr != nil {
			t.Errorf("恢复群成员失败：%v", restoreErr)
		}
	}()

	if _, err = client.Group.MemberAdd(ctx, groupID, membersToChange); err != nil {
		t.Fatal(err)
	}
	members, err := client.Group.Members(ctx, groupID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGroupMember(members, account) {
		t.Fatalf("拉人后未找到群成员 %s：%#v", account, members)
	}

	if _, err = client.Group.MemberRemove(ctx, groupID, membersToChange); err != nil {
		t.Fatal(err)
	}
	members, err = client.Group.Members(ctx, groupID)
	if err != nil {
		t.Fatal(err)
	}
	if hasGroupMember(members, account) {
		t.Fatalf("移除后仍存在群成员 %s：%#v", account, members)
	}

	if _, err = client.Group.MemberAdd(ctx, groupID, membersToChange); err != nil {
		t.Fatal(err)
	}
	restored = true
	members, err = client.Group.Members(ctx, groupID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGroupMember(members, account) {
		t.Fatalf("恢复后未找到群成员 %s：%#v", account, members)
	}
	if _, err = client.IM.SendText(ctx, tuitui.SendIMTextOptions{
		To: client.To.Group(groupID), Text: "Go SDK 群测试结束",
	}); err != nil {
		t.Fatal(err)
	}
}

func hasGroupListItem(groups []tuitui.GroupListItem, groupID string) bool {
	for _, group := range groups {
		if group.GroupID == groupID {
			return true
		}
	}
	return false
}

func hasGroupMember(members []tuitui.GroupMember, account string) bool {
	for _, member := range members {
		if member.Account == account {
			return true
		}
	}
	return false
}
