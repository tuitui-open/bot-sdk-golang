package tuitui

import "context"

// UserSearchResult 完整保留服务端返回的用户字段。
type UserSearchResult map[string]interface{}

// UserAPI 提供用户搜索接口。
type UserAPI struct{ http *httpAPI }

// Search 按原始关键词搜索用户。
func (api *UserAPI) Search(ctx context.Context, keyword string) ([]UserSearchResult, error) {
	response, err := api.http.post(ctx, "/user/search", map[string]interface{}{"keyword": keyword})
	if err != nil {
		return nil, err
	}
	values, ok := response["users"].([]interface{})
	if !ok {
		return nil, &APIError{Endpoint: "/user/search", Message: "/user/search returned invalid users", Response: response}
	}
	users := make([]UserSearchResult, 0, len(values))
	for _, value := range values {
		user, ok := value.(map[string]interface{})
		if !ok || user == nil {
			return nil, &APIError{Endpoint: "/user/search", Message: "/user/search returned invalid users", Response: response}
		}
		users = append(users, UserSearchResult(user))
	}
	return users, nil
}
