package tuitui

import "context"

type GetChatRecordOptions struct {
	StartTime       string
	EndTime         string
	RelativeTime    string
	Limit           int
	Cursor          string
	OrderAsc        *bool
	CompactForAgent bool
}

type recordsAPI struct{ http *httpAPI }

func (r *recordsAPI) getPrivateHistory(ctx context.Context, user string, options *GetChatRecordOptions) (APIResponse, error) {
	return r.history(ctx, "/message/single/sync", "user", user, options)
}
func (r *recordsAPI) getGroupHistory(ctx context.Context, groupID string, options *GetChatRecordOptions) (APIResponse, error) {
	return r.history(ctx, "/message/group/sync", "group_id", groupID, options)
}
func (r *recordsAPI) history(ctx context.Context, endpoint, key, value string, options *GetChatRecordOptions) (APIResponse, error) {
	resolved := GetChatRecordOptions{Cursor: "0"}
	if options != nil {
		resolved = *options
		if resolved.Cursor == "" {
			resolved.Cursor = "0"
		}
	}
	payload := map[string]interface{}{key: value, "cursor": resolved.Cursor}
	if resolved.RelativeTime != "" {
		payload["relative_time"] = resolved.RelativeTime
	} else {
		if resolved.StartTime != "" {
			payload["start_time"] = resolved.StartTime
		}
		if resolved.EndTime != "" {
			payload["end_time"] = resolved.EndTime
		}
	}
	if resolved.Limit != 0 {
		payload["limit"] = resolved.Limit
	}
	if resolved.OrderAsc != nil {
		payload["order_asc"] = *resolved.OrderAsc
	}
	response, err := r.http.post(ctx, endpoint, payload)
	if err != nil || !resolved.CompactForAgent {
		return response, err
	}
	return compactChatRecord(response), nil
}

func compactChatRecord(value APIResponse) APIResponse {
	result := APIResponse{"errcode": value["errcode"], "errmsg": value["errmsg"], "cursor": value["cursor"], "has_more": value["has_more"], "current_time": value["time"]}
	raw, _ := value["msgs"].([]interface{})
	messages := make([]interface{}, 0, len(raw))
	for _, item := range raw {
		message, _ := item.(map[string]interface{})
		data, _ := message["data"].(map[string]interface{})
		compact := map[string]interface{}{}
		for key, item := range data {
			if key != "at" && key != "msgid" && key != "group_id" && key != "group_name" {
				compact[key] = item
			}
		}
		compact["user_account"] = message["user_account"]
		compact["user_name"] = message["user_name"]
		compact["msg_time"] = formatTimestamp(message["timestamp"])
		messages = append(messages, compact)
	}
	result["msgs"] = messages
	return result
}
