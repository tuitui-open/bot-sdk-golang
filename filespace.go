package tuitui

import (
	"context"
	"fmt"
	"strings"
)

const (
	NodeTypeDir      = "1"
	NodeTypeFile     = "2"
	SpaceTypeGroup   = "1"
	SpaceTypeTeam    = "2"
	SpaceTypeChannel = "3"
)

type FileSpaceNode map[string]interface{}
type FileSpaceItem struct{ Filename, Author, URL, FileSize string }
type FileSpaceContext struct{ SpaceID, SpaceType, Source string }
type AddFileSpaceNodeOptions struct{ SpaceID, SpaceType, NodeType, Name, ParentID, Source, FID string }
type AddTeamFileOptions struct {
	UploadOptions
	SourcePostID string
}
type FileSpaceAPI struct {
	http     *httpAPI
	uploader *uploader
	teams    *TeamsAPI
}

func (f *FileSpaceAPI) ListNodes(ctx context.Context, context FileSpaceContext) ([]FileSpaceNode, error) {
	spaceType := firstNonEmpty(context.SpaceType, SpaceTypeTeam)
	body, err := f.http.post(ctx, "/file_space/node/list", map[string]interface{}{"space_id": context.SpaceID, "space_type": spaceType})
	if err != nil {
		return nil, err
	}
	datas, _ := body["datas"].(map[string]interface{})
	raw, _ := datas["list"].([]interface{})
	result := make([]FileSpaceNode, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(map[string]interface{}); ok {
			result = append(result, FileSpaceNode(value))
		}
	}
	return result, nil
}
func (f *FileSpaceAPI) ListFiles(ctx context.Context, context FileSpaceContext) ([]FileSpaceItem, error) {
	nodes, err := f.ListNodes(ctx, context)
	if err != nil {
		return nil, err
	}
	return FlattenFileSpaceList(nodes), nil
}
func (f *FileSpaceAPI) AddNode(ctx context.Context, options AddFileSpaceNodeOptions) (APIResponse, error) {
	payload := map[string]interface{}{"space_id": options.SpaceID, "space_type": options.SpaceType, "node_type": options.NodeType, "name": options.Name}
	if options.ParentID != "" {
		payload["parent_id"] = options.ParentID
	}
	if options.Source != "" {
		payload["source"] = options.Source
	}
	if options.FID != "" {
		payload["fid"] = options.FID
	}
	return f.http.post(ctx, "/file_space/node/add", payload)
}
func (f *FileSpaceAPI) AddFile(ctx context.Context, context FileSpaceContext, cloudPath string, source interface{}, options *UploadOptions) (APIResponse, error) {
	parts := strings.FieldsFunc(strings.TrimLeft(cloudPath, "/"), func(r rune) bool { return r == '/' })
	filename := "unnamed"
	folders := []string{}
	if len(parts) > 0 {
		filename = parts[len(parts)-1]
		folders = parts[:len(parts)-1]
	}
	parentID, err := f.ensureFolders(ctx, context, folders)
	if err != nil {
		return nil, err
	}
	resolved := UploadOptions{}
	if options != nil {
		resolved = *options
	}
	resolved.Filename = filename
	uploaded, err := f.uploader.upload(ctx, source, &resolved)
	if err != nil {
		return nil, err
	}
	return f.AddNode(ctx, AddFileSpaceNodeOptions{SpaceID: context.SpaceID, SpaceType: firstNonEmpty(context.SpaceType, SpaceTypeTeam), NodeType: NodeTypeFile, Name: filename, FID: uploaded.FID, ParentID: parentID, Source: context.Source})
}
func (f *FileSpaceAPI) ListTeamFilesByChannel(ctx context.Context, channelID string) ([]FileSpaceItem, error) {
	info, err := f.teams.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return f.ListFiles(ctx, FileSpaceContext{SpaceID: stringValue(info["team_id"]), SpaceType: SpaceTypeTeam})
}
func (f *FileSpaceAPI) AddTeamFileByChannel(ctx context.Context, channelID, cloudPath string, source interface{}, options *AddTeamFileOptions) (APIResponse, error) {
	info, err := f.teams.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, err
	}
	resolved := AddTeamFileOptions{}
	if options != nil {
		resolved = *options
	}
	return f.AddFile(ctx, FileSpaceContext{SpaceID: stringValue(info["team_id"]), SpaceType: SpaceTypeTeam, Source: resolved.SourcePostID}, cloudPath, source, &resolved.UploadOptions)
}
func (f *FileSpaceAPI) ensureFolders(ctx context.Context, context FileSpaceContext, folders []string) (string, error) {
	if len(folders) == 0 {
		return "", nil
	}
	nodes, err := f.ListNodes(ctx, context)
	if err != nil {
		return "", err
	}
	parentID := ""
	for _, name := range folders {
		var folder FileSpaceNode
		for _, node := range nodes {
			if fmt.Sprint(node["parent_id"]) == parentID && fmt.Sprint(node["node_type"]) == NodeTypeDir && fmt.Sprint(node["name"]) == name {
				folder = node
				break
			}
		}
		if folder == nil {
			result, err := f.AddNode(ctx, AddFileSpaceNodeOptions{SpaceID: context.SpaceID, SpaceType: firstNonEmpty(context.SpaceType, SpaceTypeTeam), NodeType: NodeTypeDir, Name: name, ParentID: parentID, Source: context.Source})
			if err != nil {
				return "", err
			}
			datas, _ := result["datas"].(map[string]interface{})
			nodeID := firstNonEmpty(stringValue(datas["node_id"]), stringValue(result["node_id"]))
			if nodeID == "" {
				return "", fmt.Errorf("[tuitui] failed to create folder %s", name)
			}
			folder = FileSpaceNode{"node_id": nodeID, "node_type": NodeTypeDir, "name": name, "parent_id": parentID}
			nodes = append(nodes, folder)
		}
		parentID = fmt.Sprint(folder["node_id"])
	}
	return parentID, nil
}
func FlattenFileSpaceList(list []FileSpaceNode) []FileSpaceItem {
	nodes := map[string]FileSpaceNode{}
	for _, node := range list {
		nodes[fmt.Sprint(node["node_id"])] = node
	}
	result := []FileSpaceItem{}
	for _, node := range list {
		if fmt.Sprint(node["node_type"]) != NodeTypeFile {
			continue
		}
		path := []string{fmt.Sprint(node["name"])}
		current := node
		for {
			parentID := fmt.Sprint(current["parent_id"])
			parent, ok := nodes[parentID]
			if !ok || parentID == "" || parentID == "<nil>" {
				break
			}
			path = append([]string{fmt.Sprint(parent["name"])}, path...)
			current = parent
		}
		result = append(result, FileSpaceItem{Filename: strings.Join(path, "/"), Author: fmt.Sprint(node["author_name"]), URL: fmt.Sprint(node["file_url"]), FileSize: fmt.Sprint(node["file_size"])})
	}
	return result
}
