package tuitui

// AgentEventContext 是一次执行的关联字段；构造上下文不会发起网络请求。
type AgentEventContext struct {
	SessionKey string `json:"sessionKey"`
	MessageID  string `json:"messageId"`
	RunID      string `json:"runId"`
}

// AgentEventWorkContext 保留主 Agent 或子 Agent 的完整关联字段。
type AgentEventWorkContext interface{ agentContext() }

func (AgentEventContext) agentContext() {}

type AgentEventSubagentContext struct {
	AgentEventContext
	RequesterSessionKey string `json:"requesterSessionKey"`
}
type AgentEventToolContext struct {
	Context    AgentEventWorkContext
	ToolCallID string
}
type AgentContextTarget struct {
	Type         string
	Account      string
	GroupID      string
	ChannelID    string
	PostID       string
	ParentPostID *string
}
type AgentContextOptions struct{ ChannelID string }
type BuildAgentContextOptions struct {
	Target    AgentContextTarget
	MessageID string
	ChannelID string
}
type BuildAgentSubagentContextOptions struct {
	RequesterContext AgentEventContext
	SubagentID       *string
}
type AgentEventMessage map[string]interface{}
type AgentEventLLMUsage struct {
	Input      float64                `json:"input"`
	Output     float64                `json:"output"`
	CacheRead  float64                `json:"cacheRead"`
	CacheWrite *float64               `json:"cacheWrite,omitempty"`
	Total      float64                `json:"total"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentEndStatus string

const (
	AgentEndDone     AgentEndStatus = "done"
	AgentEndFailed   AgentEndStatus = "failed"
	AgentEndCanceled AgentEndStatus = "canceled"
	AgentEndTimeout  AgentEndStatus = "timeout"
)

type AgentSubagentOutcome string

const (
	AgentSubagentOK        AgentSubagentOutcome = "ok"
	AgentSubagentError     AgentSubagentOutcome = "error"
	AgentSubagentCancelled AgentSubagentOutcome = "cancelled"
)

// AgentEvent 由八种具体事件限定，事件名称由类型决定。
type AgentEvent interface{ agentEvent() }

const AgentEventMessageReceived = "message_received"

type AgentEventMessageReceivedData struct {
	Content    string                 `json:"content"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentMessageReceivedEvent struct {
	Context AgentEventContext
	Data    AgentEventMessageReceivedData
}

func (AgentMessageReceivedEvent) agentEvent() {}

const AgentEventLLMInput = "llm_input"

type AgentEventLLMInputData struct {
	SessionID       string                 `json:"sessionId,omitempty"`
	Provider        string                 `json:"provider,omitempty"`
	Model           string                 `json:"model"`
	SystemPrompt    string                 `json:"systemPrompt,omitempty"`
	Prompt          string                 `json:"prompt"`
	HistoryMessages []AgentEventMessage    `json:"historyMessages,omitempty"`
	ImagesCount     *float64               `json:"imagesCount,omitempty"`
	Extensions      map[string]interface{} `json:"-"`
}
type AgentLLMInputEvent struct {
	Context AgentEventWorkContext
	Data    AgentEventLLMInputData
}

func (AgentLLMInputEvent) agentEvent() {}

const AgentEventLLMOutput = "llm_output"

type AgentEventLLMOutputData struct {
	SessionID      string                 `json:"sessionId,omitempty"`
	Provider       string                 `json:"provider,omitempty"`
	Model          string                 `json:"model,omitempty"`
	AssistantTexts []string               `json:"assistantTexts"`
	Thinking       interface{}            `json:"thinking"`
	LastAssistant  AgentEventMessage      `json:"lastAssistant,omitempty"`
	Usage          AgentEventLLMUsage     `json:"usage"`
	Extensions     map[string]interface{} `json:"-"`
}
type AgentLLMOutputEvent struct {
	Context AgentEventWorkContext
	Data    AgentEventLLMOutputData
}

func (AgentLLMOutputEvent) agentEvent() {}

const AgentEventBeforeToolCall = "before_tool_call"

type AgentEventBeforeToolCallData struct {
	ToolName   string                 `json:"toolName"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentBeforeToolCallEvent struct {
	Context AgentEventToolContext
	Data    AgentEventBeforeToolCallData
}

func (AgentBeforeToolCallEvent) agentEvent() {}

const AgentEventAfterToolCall = "after_tool_call"

type AgentEventAfterToolCallData struct {
	ToolName   string                 `json:"toolName"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Result     interface{}            `json:"result"`
	Error      *string                `json:"error,omitempty"`
	DurationMS *float64               `json:"durationMs,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentAfterToolCallEvent struct {
	Context AgentEventToolContext
	Data    AgentEventAfterToolCallData
}

func (AgentAfterToolCallEvent) agentEvent() {}

const AgentEventSubagentSpawned = "subagent_spawned"

type AgentEventSubagentSpawnedData struct {
	AgentID    string                 `json:"agentId,omitempty"`
	Label      string                 `json:"label,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentSubagentSpawnedEvent struct {
	Context AgentEventSubagentContext
	Data    AgentEventSubagentSpawnedData
}

func (AgentSubagentSpawnedEvent) agentEvent() {}

const AgentEventSubagentEnded = "subagent_ended"

type AgentEventSubagentEndedData struct {
	AgentID    string                 `json:"agentId,omitempty"`
	Outcome    AgentSubagentOutcome   `json:"outcome"`
	Reason     string                 `json:"reason,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentSubagentEndedEvent struct {
	Context AgentEventSubagentContext
	Data    AgentEventSubagentEndedData
}

func (AgentSubagentEndedEvent) agentEvent() {}

const AgentEventEnd = "agent_end"

type AgentEventEndData struct {
	Messages   []AgentEventMessage    `json:"messages,omitempty"`
	Success    bool                   `json:"success"`
	Status     AgentEndStatus         `json:"status"`
	Reason     string                 `json:"reason,omitempty"`
	Error      *string                `json:"error,omitempty"`
	DurationMS *float64               `json:"durationMs,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentEndEvent struct {
	Context AgentEventContext
	Data    AgentEventEndData
}

func (AgentEndEvent) agentEvent() {}
