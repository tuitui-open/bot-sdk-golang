package tuitui

import "context"

type RequestOptions struct{ Method string }

type Client struct {
	config    resolvedConfig
	http      *httpAPI
	uploader  *uploader
	IM        *IMAPI
	To        ToAPI
	Teams     *TeamsAPI
	File      *FileAPI
	FileSpace *FileSpaceAPI
	Group     *GroupAPI
	Event     *EventAPI
	Property  *PropertyAPI
	Agent     *AgentAPI
}

func NewClient(appID, appSecret string, options *ClientOptions) *Client {
	config := resolveConfig(appID, appSecret, options)
	httpClient := &httpAPI{config: config}
	uploader := &uploader{http: httpClient, config: config}
	records := &recordsAPI{http: httpClient}
	teams := &TeamsAPI{http: httpClient, uploader: uploader}
	client := &Client{config: config, http: httpClient, uploader: uploader}
	client.Agent = &AgentAPI{http: httpClient, config: config}
	client.Agent.Report = &AgentReporter{api: client.Agent}
	client.To = ToAPI{}
	client.IM = &IMAPI{http: httpClient, uploader: uploader, records: records}
	client.Teams = teams
	client.File = &FileAPI{http: httpClient, uploader: uploader}
	client.FileSpace = &FileSpaceAPI{http: httpClient, uploader: uploader, teams: teams}
	client.Group = &GroupAPI{http: httpClient}
	client.Event = &EventAPI{config: config, teams: teams}
	client.Property = &PropertyAPI{http: httpClient}
	return client
}

func (c *Client) Request(ctx context.Context, endpoint string, payload interface{}, options *RequestOptions) (APIResponse, error) {
	if options != nil && options.Method == "GET" {
		return c.http.get(ctx, endpoint)
	}
	return c.http.post(ctx, endpoint, payload)
}
