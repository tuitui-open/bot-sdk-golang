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
	Event     *EventAPI
	Property  *PropertyAPI
}

func NewClient(appID, appSecret string, options *ClientOptions) (*Client, error) {
	config, err := resolveConfig(appID, appSecret, options)
	if err != nil {
		return nil, err
	}
	httpClient := &httpAPI{config: config}
	uploader := &uploader{http: httpClient, config: config}
	records := &recordsAPI{http: httpClient}
	teams := &TeamsAPI{http: httpClient, uploader: uploader}
	client := &Client{config: config, http: httpClient, uploader: uploader}
	client.To = ToAPI{}
	client.IM = &IMAPI{http: httpClient, uploader: uploader, records: records}
	client.Teams = teams
	client.File = &FileAPI{uploader: uploader}
	client.FileSpace = &FileSpaceAPI{http: httpClient, uploader: uploader, teams: teams}
	client.Event = &EventAPI{config: config, teams: teams}
	client.Property = &PropertyAPI{http: httpClient}
	return client, nil
}

func (c *Client) Request(ctx context.Context, endpoint string, payload any, options *RequestOptions) (APIResponse, error) {
	if options != nil && options.Method == "GET" {
		return c.http.get(ctx, endpoint)
	}
	return c.http.post(ctx, endpoint, payload)
}
