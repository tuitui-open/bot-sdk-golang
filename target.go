package tuitui

import (
	"fmt"
	"strings"
)

type ToTarget struct {
	Accounts []string
	UIDs     []string
	GroupID  string
}

type ToAPI struct{}

func (ToAPI) Account(account string) ToTarget {
	return ToTarget{Accounts: normalizeValues([]string{account})}
}
func (ToAPI) UID(uid string) ToTarget           { return ToTarget{UIDs: normalizeValues([]string{uid})} }
func (ToAPI) Accounts(values []string) ToTarget { return ToTarget{Accounts: normalizeValues(values)} }
func (ToAPI) UIDs(values []string) ToTarget     { return ToTarget{UIDs: normalizeValues(values)} }
func (ToAPI) Group(groupID string) ToTarget     { return ToTarget{GroupID: strings.TrimSpace(groupID)} }

type resolvedTarget struct {
	kind     string
	accounts []string
	uids     []string
	groupID  string
}

func resolveTarget(target ToTarget) (resolvedTarget, error) {
	accounts := normalizeValues(target.Accounts)
	uids := normalizeValues(target.UIDs)
	groupID := strings.TrimSpace(target.GroupID)
	if groupID != "" {
		if len(accounts) > 0 || len(uids) > 0 {
			return resolvedTarget{}, fmt.Errorf("[tuitui] user and group targets cannot be mixed")
		}
		return resolvedTarget{kind: "group", groupID: groupID}, nil
	}
	if len(accounts) == 0 && len(uids) == 0 {
		return resolvedTarget{}, fmt.Errorf("[tuitui] target must contain accounts, uids, or groupID")
	}
	if len(accounts) > 100 || len(uids) > 100 {
		return resolvedTarget{}, fmt.Errorf("[tuitui] user targets exceed the limit of 100")
	}
	return resolvedTarget{kind: "users", accounts: accounts, uids: uids}, nil
}

func normalizeValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
