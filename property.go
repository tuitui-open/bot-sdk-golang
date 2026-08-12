package tuitui

import (
	"context"
	"strings"
)

type BotInfo struct {
	Name    string
	UID     string
	Account string
}

type ShortcutCommand struct {
	Name        string
	Content     string
	Description string
}

type SetShortcutCommandsOptions struct{ NoAt *bool }

type PropertyAPI struct{ http *httpAPI }

func (p *PropertyAPI) Info(ctx context.Context) (BotInfo, error) {
	response, err := p.http.get(ctx, "/prop/get")
	if err != nil {
		return BotInfo{}, err
	}
	return BotInfo{Name: strings.TrimSpace(stringValue(response["robot_name"])), UID: stringValue(response["robot_uid"]), Account: strings.TrimSpace(stringValue(response["robot_account"]))}, nil
}

func (p *PropertyAPI) SetName(ctx context.Context, name string) (APIResponse, error) {
	return p.http.post(ctx, "/name/modify", map[string]any{"name": name})
}
func (p *PropertyAPI) SetAvatar(ctx context.Context, avatar string) (APIResponse, error) {
	return p.http.post(ctx, "/avatar/modify", map[string]any{"avatar": avatar})
}
func (p *PropertyAPI) SetWebhook(ctx context.Context, value string) (APIResponse, error) {
	return p.http.post(ctx, "/webhook/modify", map[string]any{"url": value})
}
func (p *PropertyAPI) SetInteractiveURL(ctx context.Context, value string) (APIResponse, error) {
	return p.http.post(ctx, "/interactive_url/modify", map[string]any{"url": value})
}
func (p *PropertyAPI) SetShortcutCommands(ctx context.Context, commands []ShortcutCommand, options *SetShortcutCommandsOptions) (APIResponse, error) {
	items := make([]map[string]any, 0, len(commands))
	for _, command := range commands {
		items = append(items, map[string]any{"command_name": command.Name, "command_content": command.Content, "command_description": command.Description})
	}
	payload := map[string]any{"shortcut_cmds": items}
	if options != nil && options.NoAt != nil {
		payload["no_at"] = *options.NoAt
	}
	return p.http.post(ctx, "/shortcutCommand/set", payload)
}
func (p *PropertyAPI) GetShortcutCommands(ctx context.Context) ([]ShortcutCommand, error) {
	response, err := p.http.post(ctx, "/shortcutCommand/get", map[string]any{})
	if err != nil {
		return nil, err
	}
	data, _ := response["datas"].(map[string]any)
	raw, _ := data["shortcut_cmds"].([]any)
	commands := make([]ShortcutCommand, 0, len(raw))
	for _, item := range raw {
		command, _ := item.(map[string]any)
		commands = append(commands, ShortcutCommand{Name: stringValue(command["command_name"]), Content: stringValue(command["command_content"]), Description: stringValue(command["command_description"])})
	}
	return commands, nil
}
