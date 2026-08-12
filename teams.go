package tuitui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SendPostOptions struct{ TeamID, ChannelID, Text, ParentID, Tag string }
type EditPostOptions struct{ TeamID, ChannelID, PostID, Text, Tag string }
type SendPostFileOptions struct {
	TeamID, ChannelID     string
	Source                any
	Text, ParentID        string
	At                    []string
	Filename, ContentType string
}
type PostEmojiReactionOptions struct {
	TeamID, ChannelID, PostID, Emoji string
	Cancel                           bool
}
type ChannelInfo map[string]any
type TeamMember map[string]any
type GetPostTopicsOptions struct {
	TeamID        string `json:"team_id"`
	ChannelID     string `json:"channel_id"`
	Size          int    `json:"size,omitempty"`
	SortType      string `json:"sort_type,omitempty"`
	Order         string `json:"order,omitempty"`
	FromTimestamp int64  `json:"from_timestamp,omitempty"`
	EndTimestamp  int64  `json:"end_timestamp,omitempty"`
}
type PostChainItem struct {
	FromUID, PostID, Time, LastReplyTime, Name, Content string
	Properties                                          any
}

type TeamsAPI struct {
	http     *httpAPI
	uploader *uploader
}

func (t *TeamsAPI) SendPost(ctx context.Context, options SendPostOptions) (APIResponse, error) {
	if strings.TrimSpace(options.TeamID) == "" {
		return nil, fmt.Errorf("[tuitui] teamID is required")
	}
	if strings.TrimSpace(options.ChannelID) == "" {
		return nil, fmt.Errorf("[tuitui] channelID is required")
	}
	target := map[string]any{"team_id": options.TeamID, "channel_id": options.ChannelID}
	if options.ParentID != "" {
		target["parent_id"] = options.ParentID
	}
	if options.Tag != "" {
		tagID, err := t.channelPostTagID(ctx, options.ChannelID, options.Tag)
		if err != nil {
			return nil, err
		}
		target["tags"] = []string{tagID}
	}
	return t.sendMarkdown(ctx, "/message/custom/send", target, options.Text)
}
func (t *TeamsAPI) EditPost(ctx context.Context, options EditPostOptions) (APIResponse, error) {
	if options.TeamID == "" || options.ChannelID == "" || options.PostID == "" {
		return nil, fmt.Errorf("[tuitui] teamID, channelID, and postID are required")
	}
	target := map[string]any{"team_id": options.TeamID, "channel_id": options.ChannelID, "post_id": options.PostID}
	if options.Tag != "" {
		tagID, err := t.channelPostTagID(ctx, options.ChannelID, options.Tag)
		if err != nil {
			return nil, err
		}
		target["tags"] = []string{tagID}
	}
	return t.sendMarkdown(ctx, "/message/custom/modify", target, options.Text)
}
func (t *TeamsAPI) sendMarkdown(ctx context.Context, endpoint string, target map[string]any, text string) (APIResponse, error) {
	markdown := replaceMentions(text)
	hasMention := markdown != text
	payload := map[string]any{"toteams": []any{target}, "msgtype": "richtext/markdown", "richtext": map[string]any{"markdown": markdown, "delims_left": map[bool]string{true: "{{"}[hasMention], "delims_right": map[bool]string{true: "}}"}[hasMention]}}
	response, err := t.http.post(ctx, endpoint, payload)
	if err == nil || !hasMention {
		return response, err
	}
	payload["richtext"] = map[string]any{"markdown": text, "delims_left": "", "delims_right": ""}
	return t.http.post(ctx, endpoint, payload)
}
func (t *TeamsAPI) SendFile(ctx context.Context, options SendPostFileOptions) (APIResponse, error) {
	uploaded, err := t.uploader.upload(ctx, options.Source, &UploadOptions{Filename: options.Filename, ContentType: options.ContentType})
	if err != nil {
		return nil, err
	}
	target := map[string]any{"team_id": options.TeamID, "channel_id": options.ChannelID}
	if options.ParentID != "" {
		target["parent_id"] = options.ParentID
	}
	fileMarkdown := fmt.Sprintf("[%s]({{tuitui_file \"%s\"}})", uploaded.Filename, uploaded.FID)
	if uploaded.MediaType == "image" {
		fileMarkdown = fmt.Sprintf("![]({{tuitui_image \"%s\"}})", uploaded.FID)
	}
	markdown := fileMarkdown
	if options.Text != "" {
		markdown = options.Text + "\n\n" + fileMarkdown
	}
	return t.http.post(ctx, "/message/custom/send", map[string]any{"toteams": []any{target}, "msgtype": "richtext/markdown", "at": options.At, "richtext": map[string]any{"markdown": markdown, "delims_left": "{{", "delims_right": "}}"}})
}
func (t *TeamsAPI) EmojiReaction(ctx context.Context, options PostEmojiReactionOptions) (APIResponse, error) {
	return t.http.post(ctx, "/message/custom/modify", map[string]any{"toteams": []any{map[string]any{"team_id": options.TeamID, "channel_id": options.ChannelID, "parent_id": "", "post_id": options.PostID}}, "msgtype": "emoji_reaction", "emoji_reaction": map[string]any{"emoji": options.Emoji, "cancel": options.Cancel}})
}
func (t *TeamsAPI) GetChannelInfo(ctx context.Context, channelID string) (ChannelInfo, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, fmt.Errorf("[tuitui] channelID is required")
	}
	body, err := t.http.post(ctx, "/teams/channel/info", map[string]any{"channel_id": channelID})
	if err != nil {
		return nil, err
	}
	datas, _ := body["datas"].(map[string]any)
	info, _ := datas["info"].(map[string]any)
	if stringValue(info["team_id"]) == "" {
		return nil, &APIError{Message: fmt.Sprintf("/teams/channel/info returned invalid channel info for channel %s: team_id is missing", channelID), Endpoint: "/teams/channel/info", Response: body}
	}
	return ChannelInfo(info), nil
}
func (t *TeamsAPI) loadChannelPostTags(ctx context.Context, channelID string) ([]map[string]any, error) {
	body, err := t.http.post(ctx, "/teams/channel/postTag/list", map[string]any{"channel_id": channelID})
	if err != nil {
		return nil, err
	}
	datas, _ := body["datas"].(map[string]any)
	raw, _ := datas["tags"].([]any)
	result := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(map[string]any); ok {
			result = append(result, value)
		}
	}
	return result, nil
}
func (t *TeamsAPI) channelPostTagID(ctx context.Context, channelID, tag string) (string, error) {
	tags, err := t.loadChannelPostTags(ctx, channelID)
	if err != nil {
		return "", err
	}
	available := []string{}
	for _, item := range tags {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name != "" {
			available = append(available, name)
		}
		if strings.EqualFold(name, strings.TrimSpace(tag)) {
			id := fmt.Sprint(item["tag_id"])
			if id != "" {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("[tuitui] channel post tag not found: %s. Available tags: %s", tag, strings.Join(available, ", "))
}
func (t *TeamsAPI) GetChannelPostTags(ctx context.Context, channelID string) ([]string, error) {
	tags, err := t.loadChannelPostTags(ctx, channelID)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, item := range tags {
		if name := strings.TrimSpace(fmt.Sprint(item["name"])); name != "" {
			result = append(result, name)
		}
	}
	return result, nil
}
func (t *TeamsAPI) GetMembers(ctx context.Context, teamID string) ([]TeamMember, error) {
	body, err := t.http.post(ctx, "/teams/member/list", map[string]any{"team_id": teamID})
	if err != nil {
		return nil, err
	}
	datas, _ := body["datas"].(map[string]any)
	raw, _ := datas["members"].([]any)
	result := make([]TeamMember, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(map[string]any); ok {
			result = append(result, TeamMember(value))
		}
	}
	return result, nil
}
func (t *TeamsAPI) GetMemberNames(ctx context.Context, teamID string) (string, error) {
	members, err := t.GetMembers(ctx, teamID)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(members))
	for _, member := range members {
		names = append(names, fmt.Sprint(member["name"]))
	}
	return strings.Join(names, "\n"), nil
}
func (t *TeamsAPI) GetAnnouncement(ctx context.Context, channelID string) (any, error) {
	info, err := t.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return info["announcement"], nil
}
func (t *TeamsAPI) GetPostChain(ctx context.Context, teamID, channelID, postID string) (APIResponse, error) {
	return t.http.post(ctx, "/teams/post/chain", map[string]any{"team_id": teamID, "channel_id": channelID, "post_id": postID})
}
func (t *TeamsAPI) GetPostChainForAgent(ctx context.Context, teamID, channelID, postID string) ([]PostChainItem, error) {
	body, err := t.GetPostChain(ctx, teamID, channelID, postID)
	if err != nil {
		return nil, err
	}
	return compactPostChain(body["datas"]), nil
}
func (t *TeamsAPI) GetSharedPost(ctx context.Context, shareID string) (APIResponse, error) {
	if strings.TrimSpace(shareID) == "" {
		return nil, fmt.Errorf("[tuitui] shareID is required")
	}
	return t.http.post(ctx, "/teams/share/post", map[string]any{"share_id": shareID})
}
func (t *TeamsAPI) GetSharedPostForAgent(ctx context.Context, shareID string) (string, error) {
	body, err := t.GetSharedPost(ctx, shareID)
	if err != nil {
		return "", err
	}
	return postChainText(compactPostChain(body["datas"])), nil
}
func (t *TeamsAPI) GetPostTopics(ctx context.Context, options GetPostTopicsOptions) (APIResponse, error) {
	return t.http.post(ctx, "/teams/post/topic/list", options)
}

func (t *TeamsAPI) GetChannelPosts(ctx context.Context, channelID string, options *GetChatRecordOptions) (APIResponse, error) {
	resolved := GetChatRecordOptions{}
	if options != nil {
		resolved = *options
	}
	if resolved.Cursor != "" && resolved.Cursor != "0" && resolved.RelativeTime == "" && resolved.StartTime == "" {
		return nil, fmt.Errorf("cursor must be used with relativeTime or startTime")
	}
	info, err := t.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, err
	}
	pageSize := resolved.Limit
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	payload := GetPostTopicsOptions{TeamID: stringValue(info["team_id"]), ChannelID: channelID, Size: pageSize, SortType: "reply", Order: "asc"}
	if resolved.RelativeTime != "" {
		if start, end, ok := parseRelativeTime(resolved.RelativeTime, time.Now()); ok {
			payload.FromTimestamp = start.UnixMilli()
			payload.EndTimestamp = end.UnixMilli()
		}
	} else {
		if value, err := time.Parse(time.RFC3339, resolved.StartTime); err == nil {
			payload.FromTimestamp = value.UnixMilli()
		}
		if value, err := time.Parse(time.RFC3339, resolved.EndTime); err == nil {
			payload.EndTimestamp = value.UnixMilli()
		}
	}
	if resolved.Cursor != "" && resolved.Cursor != "0" {
		payload.FromTimestamp, _ = strconv.ParseInt(resolved.Cursor, 10, 64)
	}
	body, err := t.GetPostTopics(ctx, payload)
	if err != nil {
		return nil, err
	}
	datas, _ := body["datas"].(map[string]any)
	posts, _ := datas["post_list"].([]any)
	threads := make([]string, 0, len(posts))
	lastTimestamp := ""
	for _, post := range posts {
		chain := compactPostChain(post)
		if len(chain) == 0 {
			continue
		}
		threads = append(threads, postChainText(chain))
		lastTimestamp = chain[0].LastReplyTime
	}
	hasMore := len(posts) >= pageSize
	cursor := ""
	if hasMore {
		if timestamp, err := strconv.ParseInt(lastTimestamp, 10, 64); err == nil {
			cursor = strconv.FormatInt(timestamp+1, 10)
		}
	}
	return APIResponse{"errcode": body["errcode"], "errmsg": body["errmsg"], "cursor": cursor, "has_more": hasMore, "time": body["time"], "subject": fmt.Sprint(info["name"]), "threads": threads}, nil
}

func compactPostChain(value any) []PostChainItem {
	item, _ := value.(map[string]any)
	topic, _ := item["topic"].(map[string]any)
	result := []PostChainItem{convertPost(topic, true)}
	raw, _ := item["reply_list"].([]any)
	for index := len(raw) - 1; index >= 0; index-- {
		post, _ := raw[index].(map[string]any)
		result = append(result, convertPost(post, false))
	}
	return result
}
func convertPost(post map[string]any, main bool) PostChainItem {
	last := ""
	if main {
		last = fmt.Sprint(post["last_reply_time"])
	}
	return PostChainItem{FromUID: fmt.Sprint(post["from_uid"]), PostID: fmt.Sprint(post["post_id"]), Time: fmt.Sprint(post["create_time"]), LastReplyTime: last, Name: fmt.Sprint(post["from_name"]), Content: fmt.Sprint(post["content"]), Properties: post["properties"]}
}
func postChainText(posts []PostChainItem) string {
	lines := []string{"以下为一个独立的帖子讨论串，包含主贴和回帖"}
	for index, post := range posts {
		kind := "[讨论回帖]"
		if index == 0 {
			kind = "[讨论主贴]"
		}
		lines = append(lines, kind, "发言人: "+post.Name, "时间: "+formatTimestamp(post.Time), "内容: "+post.Content, "")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
func replaceMentions(text string) string {
	return mentionPattern.ReplaceAllStringFunc(text, func(match string) string {
		at := strings.LastIndex(match, "@")
		if at < 0 {
			return match
		}
		return match[:at] + `{{tuitui_at "` + match[at+1:] + `"}}`
	})
}
