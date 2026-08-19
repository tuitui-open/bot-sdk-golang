package tuitui

import "context"

type CreateGroupOptions struct {
	Name    string
	Owner   string
	Members []string
}

type CreateGroupResult struct {
	GroupID string
}

type UserIsInOptions struct {
	User   string
	Groups []string
}

type GroupInfo struct {
	ID       string
	Name     string
	Announce string
}

type GroupListItem struct {
	GroupID string
	Name    string
}

type GroupMemberBotInfo struct {
	Desc string
}

type GroupMember struct {
	UID     string
	Account string
	Name    string
	Role    string
	IsBot   bool
	BotInfo *GroupMemberBotInfo
	Dept    []string
}

// GroupAPI 提供群创建、成员管理和群信息查询接口。
type GroupAPI struct{ http *httpAPI }

// Create 创建群聊。群成员最多 100 人；服务端限制每个机器人每天最多调用 100 次。
func (api *GroupAPI) Create(ctx context.Context, options CreateGroupOptions) (CreateGroupResult, error) {
	response, err := api.http.post(ctx, "/group/create", map[string]interface{}{
		"name":    options.Name,
		"owner":   options.Owner,
		"members": options.Members,
	})
	if err != nil {
		return CreateGroupResult{}, err
	}
	return CreateGroupResult{GroupID: stringValue(response["group_id"])}, nil
}

// MemberAdd 向群聊添加成员。
func (api *GroupAPI) MemberAdd(ctx context.Context, groupID string, members []string) (APIResponse, error) {
	return api.http.post(ctx, "/group/member/add", map[string]interface{}{
		"group_id": groupID,
		"members":  members,
	})
}

// MemberRemove 从群聊移除成员。
func (api *GroupAPI) MemberRemove(ctx context.Context, groupID string, members []string) (APIResponse, error) {
	return api.http.post(ctx, "/group/member/remove", map[string]interface{}{
		"group_id": groupID,
		"members":  members,
	})
}

// RobotIn 获取当前机器人所在的群聊列表。
func (api *GroupAPI) RobotIn(ctx context.Context) ([]GroupListItem, error) {
	response, err := api.http.get(ctx, "/group/robot/in")
	if err != nil {
		return nil, err
	}
	return groupList(response["groups"]), nil
}

// UserIsIn 从指定群列表中返回用户和机器人共同所在的群，返回值不是布尔值。
func (api *GroupAPI) UserIsIn(ctx context.Context, options UserIsInOptions) ([]GroupListItem, error) {
	response, err := api.http.post(ctx, "/group/user/isin", map[string]interface{}{
		"user":   options.User,
		"groups": options.Groups,
	})
	if err != nil {
		return nil, err
	}
	return groupList(response["groups"]), nil
}

// Members 获取群成员列表，要求机器人已经在群内。
func (api *GroupAPI) Members(ctx context.Context, groupID string) ([]GroupMember, error) {
	response, err := api.http.post(ctx, "/group/members", map[string]interface{}{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	values, _ := response["members"].([]interface{})
	members := make([]GroupMember, 0, len(values))
	for _, value := range values {
		item, _ := value.(map[string]interface{})
		member := GroupMember{
			UID:     stringValue(item["uid"]),
			Account: stringValue(item["account"]),
			Name:    stringValue(item["name"]),
			Role:    stringValue(item["role"]),
			IsBot:   item["is_bot"] == true,
			Dept:    stringSlice(item["dept"]),
		}
		if botInfo, ok := item["bot_info"].(map[string]interface{}); ok {
			member.BotInfo = &GroupMemberBotInfo{Desc: stringValue(botInfo["desc"])}
		}
		members = append(members, member)
	}
	return members, nil
}

// Info 获取群名称和公告等信息，要求机器人已经在群内。
func (api *GroupAPI) Info(ctx context.Context, groupID string) (GroupInfo, error) {
	response, err := api.http.post(ctx, "/group/info", map[string]interface{}{"group_id": groupID})
	if err != nil {
		return GroupInfo{}, err
	}
	info, _ := response["group_info"].(map[string]interface{})
	return GroupInfo{
		ID:       stringValue(info["id"]),
		Name:     stringValue(info["name"]),
		Announce: stringValue(info["announce"]),
	}, nil
}

func stringSlice(value interface{}) []string {
	values, _ := value.([]interface{})
	result := make([]string, 0, len(values))
	for _, item := range values {
		result = append(result, stringValue(item))
	}
	return result
}

func groupList(value interface{}) []GroupListItem {
	values, _ := value.([]interface{})
	groups := make([]GroupListItem, 0, len(values))
	for _, value := range values {
		item, _ := value.(map[string]interface{})
		groups = append(groups, GroupListItem{
			GroupID: stringValue(item["group_id"]),
			Name:    stringValue(item["name"]),
		})
	}
	return groups
}
