package tuitui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type Page struct {
	Title        string            `json:"title"`
	Summary      string            `json:"summary,omitempty"`
	Content      string            `json:"content"`
	Image        string            `json:"image,omitempty"`
	Format       string            `json:"format,omitempty"`
	Privilege    string            `json:"privilege,omitempty"`
	DelimsLeft   string            `json:"delims_left,omitempty"`
	DelimsRight  string            `json:"delims_right,omitempty"`
	KV           map[string]string `json:"kv,omitempty"`
	DefaultValue string            `json:"default_value,omitempty"`
	Debug        bool              `json:"debug,omitempty"`
}
type Link struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Image   string `json:"image,omitempty"`
}
type InteractiveMessage map[string]interface{}

type MixedItem struct {
	Type        string
	Text        string
	Source      interface{}
	Filename    string
	ContentType string
}
type SendIMTextOptions struct {
	To                 ToTarget
	Text               string
	ReferenceMessageID string
	At                 []string
}
type SendIMMixedOptions struct {
	To    ToTarget
	Items []MixedItem
	At    []string
}
type SendIMPageOptions struct {
	To   ToTarget
	Page Page
}
type SendIMLinkOptions struct {
	To   ToTarget
	Link Link
}
type SendIMFileOptions struct {
	To          ToTarget
	Source      interface{}
	At          []string
	Filename    string
	ContentType string
}
type SendIMInteractiveOptions struct {
	To          ToTarget
	Interactive InteractiveMessage
}
type ModifyIMTextOptions struct {
	To          ToTarget
	MessageID   string
	Text        string
	WithoutPush *bool
}
type ModifyIMInteractiveOptions struct {
	To          ToTarget
	MessageID   string
	Interactive InteractiveMessage
}
type IMEmojiReactionOptions struct {
	To               ToTarget
	MessageID, Emoji string
	Cancel           bool
}

type IMAPI struct {
	http     *httpAPI
	uploader *uploader
	records  *recordsAPI
}

func (i *IMAPI) SendText(ctx context.Context, options SendIMTextOptions) (APIResponse, error) {
	payload, group, _, err := imTargetPayload(options.To, options.At)
	if err != nil {
		return nil, err
	}
	at := options.At
	if at == nil && group {
		at = extractMentions(options.Text)
	}
	text := map[string]interface{}{"content": options.Text}
	if options.ReferenceMessageID != "" {
		text["reference_msgid"] = options.ReferenceMessageID
	}
	payload["msgtype"] = "text"
	payload["at"] = at
	payload["text"] = text
	return i.http.post(ctx, "/message/custom/send", payload)
}
func (i *IMAPI) SendMixed(ctx context.Context, options SendIMMixedOptions) (APIResponse, error) {
	payload, _, _, err := imTargetPayload(options.To, options.At)
	if err != nil {
		return nil, err
	}
	items, err := i.prepareMixed(ctx, options.Items)
	if err != nil {
		return nil, err
	}
	payload["msgtype"] = "mixed"
	payload["mixed"] = items
	payload["at"] = options.At
	return i.http.post(ctx, "/message/custom/send", payload)
}
func (i *IMAPI) SendPage(ctx context.Context, options SendIMPageOptions) (APIResponse, error) {
	payload, _, _, err := imTargetPayload(options.To, nil)
	if err != nil {
		return nil, err
	}
	payload["msgtype"] = "page"
	payload["page"] = options.Page
	return i.http.post(ctx, "/message/custom/send", payload)
}
func (i *IMAPI) SendLink(ctx context.Context, options SendIMLinkOptions) (APIResponse, error) {
	payload, _, _, err := imTargetPayload(options.To, nil)
	if err != nil {
		return nil, err
	}
	payload["msgtype"] = "link"
	payload["link"] = options.Link
	return i.http.post(ctx, "/message/custom/send", payload)
}
func (i *IMAPI) SendFile(ctx context.Context, options SendIMFileOptions) (APIResponse, error) {
	payload, _, _, err := imTargetPayload(options.To, options.At)
	if err != nil {
		return nil, err
	}
	uploaded, err := i.uploader.upload(ctx, options.Source, &UploadOptions{Filename: options.Filename, ContentType: options.ContentType})
	if err != nil {
		return nil, err
	}
	payload["at"] = options.At
	if uploaded.MediaType == "image" {
		payload["msgtype"] = "image"
		payload["image"] = map[string]interface{}{"media_id": uploaded.FID}
	} else {
		payload["msgtype"] = "attachment"
		payload["attachment"] = map[string]interface{}{"media_id": uploaded.FID}
	}
	return i.http.post(ctx, "/message/custom/send", payload)
}
func (i *IMAPI) SendInteractive(ctx context.Context, options SendIMInteractiveOptions) (APIResponse, error) {
	payload, _, count, err := imTargetPayload(options.To, nil)
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, fmt.Errorf("[tuitui] interactive messages require exactly one target")
	}
	payload["msgtype"] = "interactive"
	payload["interactive"] = options.Interactive
	response, err := i.http.post(ctx, "/message/custom/send", payload)
	if err != nil {
		return nil, err
	}
	if response["msgid"] == nil {
		if msgids, ok := response["msgids"].([]interface{}); ok && len(msgids) > 0 {
			if first, ok := msgids[0].(map[string]interface{}); ok {
				response["msgid"] = first["msgid"]
			}
		}
	}
	return response, nil
}
func (i *IMAPI) ModifyText(ctx context.Context, options ModifyIMTextOptions) (APIResponse, error) {
	payload, err := editableTargetPayload(options.To, options.MessageID)
	if err != nil {
		return nil, err
	}
	payload["msgtype"] = "text"
	payload["text"] = map[string]interface{}{"content": options.Text}
	if options.WithoutPush != nil {
		payload["without_push"] = *options.WithoutPush
	}
	return i.http.post(ctx, "/message/custom/modify", payload)
}
func (i *IMAPI) ModifyInteractive(ctx context.Context, options ModifyIMInteractiveOptions) (APIResponse, error) {
	payload, err := editableTargetPayload(options.To, options.MessageID)
	if err != nil {
		return nil, err
	}
	payload["msgtype"] = "interactive"
	payload["interactive"] = options.Interactive
	return i.http.post(ctx, "/message/custom/modify", payload)
}
func (i *IMAPI) EmojiReaction(ctx context.Context, options IMEmojiReactionOptions) (APIResponse, error) {
	payload, err := editableTargetPayload(options.To, options.MessageID)
	if err != nil {
		return nil, err
	}
	payload["msgtype"] = "emoji_reaction"
	payload["emoji_reaction"] = map[string]interface{}{"emoji": options.Emoji, "cancel": options.Cancel}
	return i.http.post(ctx, "/message/custom/modify", payload)
}
func (i *IMAPI) GetHistory(ctx context.Context, target ToTarget, options *GetChatRecordOptions) (APIResponse, error) {
	resolved, err := resolveEditableTarget(target)
	if err != nil {
		return nil, err
	}
	if resolved.kind == "account" {
		return i.records.getPrivateHistory(ctx, resolved.value, options)
	}
	return i.records.getGroupHistory(ctx, resolved.value, options)
}

func (i *IMAPI) prepareMixed(ctx context.Context, items []MixedItem) ([]map[string]string, error) {
	if len(items) < 1 || len(items) > 10 {
		return nil, fmt.Errorf("[tuitui] mixed items must contain between 1 and 10 entries")
	}
	textLength := 0
	for _, item := range items {
		if item.Type == "text" {
			textLength += len([]rune(item.Text))
		}
	}
	if textLength > 50000 {
		return nil, fmt.Errorf("[tuitui] mixed text must not exceed 50000 characters")
	}
	result := make([]map[string]string, 0, len(items))
	for _, item := range items {
		if item.Type == "text" {
			result = append(result, map[string]string{"type": "text", "value": item.Text})
			continue
		}
		if item.Type != "image" {
			return nil, fmt.Errorf("[tuitui] unsupported mixed item type %q", item.Type)
		}
		uploaded, err := i.uploader.upload(ctx, item.Source, &UploadOptions{Filename: item.Filename, ContentType: item.ContentType})
		if err != nil {
			return nil, err
		}
		if uploaded.MediaType != "image" {
			return nil, fmt.Errorf("[tuitui] mixed messages only support JPG/JPEG, PNG, and GIF images: %s", uploaded.Filename)
		}
		result = append(result, map[string]string{"type": "image", "value": uploaded.FID})
	}
	return result, nil
}

func imTargetPayload(target ToTarget, at []string) (map[string]interface{}, bool, int, error) {
	resolved, err := resolveTarget(target)
	if err != nil {
		return nil, false, 0, err
	}
	if resolved.kind == "group" {
		return map[string]interface{}{"togroups": []string{resolved.groupID}}, true, 1, nil
	}
	if len(at) > 0 {
		return nil, false, 0, fmt.Errorf("[tuitui] at is only supported for group messages")
	}
	payload := map[string]interface{}{}
	if len(resolved.accounts) > 0 {
		payload["tousers"] = resolved.accounts
	}
	if len(resolved.uids) > 0 {
		payload["touids"] = resolved.uids
	}
	return payload, false, len(resolved.accounts) + len(resolved.uids), nil
}

type editableTarget struct{ kind, value string }

func resolveEditableTarget(target ToTarget) (editableTarget, error) {
	resolved, err := resolveTarget(target)
	if err != nil {
		return editableTarget{}, err
	}
	if resolved.kind == "group" {
		return editableTarget{"group", resolved.groupID}, nil
	}
	if len(resolved.uids) > 0 {
		return editableTarget{}, fmt.Errorf("[tuitui] uid targets do not support this operation")
	}
	if len(resolved.accounts) != 1 {
		return editableTarget{}, fmt.Errorf("[tuitui] this operation requires exactly one account")
	}
	return editableTarget{"account", resolved.accounts[0]}, nil
}
func editableTargetPayload(target ToTarget, messageID string) (map[string]interface{}, error) {
	if strings.TrimSpace(messageID) == "" {
		return nil, fmt.Errorf("[tuitui] messageID is required")
	}
	resolved, err := resolveEditableTarget(target)
	if err != nil {
		return nil, err
	}
	if resolved.kind == "account" {
		return map[string]interface{}{"tousers": []map[string]string{{"user": resolved.value, "msgid": messageID}}}, nil
	}
	return map[string]interface{}{"togroups": []map[string]string{{"group": resolved.value, "msgid": messageID}}}, nil
}

var mentionPattern = regexp.MustCompile(`(?:^|[\s\r\n　、。，！？…])@([^\s]+)`)

func extractMentions(text string) []string {
	matches := mentionPattern.FindAllStringSubmatch(text, -1)
	result := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			result = append(result, match[1])
		}
	}
	return result
}
