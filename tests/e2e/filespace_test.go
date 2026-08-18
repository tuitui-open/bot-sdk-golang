//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tuitui "github.com/tuitui-open/bot-sdk-golang"
)

func TestFileSpace创建并删除节点(t *testing.T) {
	requireEnv(t, "TUITUI_BOT_APPID", "TUITUI_BOT_SECRET", "TARGET_TEAM")
	client := tuitui.NewClient(os.Getenv("TUITUI_BOT_APPID"), os.Getenv("TUITUI_BOT_SECRET"), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	spaceID := os.Getenv("TARGET_TEAM")
	created, err := client.FileSpace.AddNode(ctx, tuitui.AddFileSpaceNodeOptions{
		SpaceID:   spaceID,
		SpaceType: tuitui.SpaceTypeTeam,
		NodeType:  tuitui.NodeTypeDir,
		Name:      fmt.Sprintf("sdk-go-delete-%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatal(err)
	}
	nodeID := created["datas"].(map[string]interface{})["node_id"].(string)
	deleted := false
	defer func() {
		if deleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = client.FileSpace.DeleteNode(cleanupCtx, tuitui.DeleteFileSpaceNodeOptions{
			SpaceID:   spaceID,
			SpaceType: tuitui.SpaceTypeTeam,
			NodeID:    nodeID,
		})
	}()

	_, err = client.FileSpace.DeleteNode(ctx, tuitui.DeleteFileSpaceNodeOptions{
		SpaceID:   spaceID,
		SpaceType: tuitui.SpaceTypeTeam,
		NodeID:    nodeID,
	})
	if err != nil {
		t.Fatal(err)
	}
	deleted = true

	nodes, err := client.FileSpace.ListNodes(ctx, tuitui.FileSpaceContext{
		SpaceID:   spaceID,
		SpaceType: tuitui.SpaceTypeTeam,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node["node_id"] == nodeID {
			t.Fatalf("节点删除后仍然存在：%s", nodeID)
		}
	}
}
