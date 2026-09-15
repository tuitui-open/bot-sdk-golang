package tuitui

// AgentEventContext 是一次执行的关联字段；构造上下文不会发起网络请求。
type AgentEventContext struct {
	// SessionKey 是服务端会话路由键。
	SessionKey string `json:"sessionKey"`
	// MessageID 是当前 Agent 执行关联的入站消息 ID；当前事件没有时留空。
	MessageID string `json:"messageId,omitempty"`
	// RunID 标识当前消息触发的一次 Agent 执行；当前事件没有时留空。
	RunID string `json:"runId,omitempty"`
}

// AgentEventWorkContext 保留主 Agent 或子 Agent 的完整关联字段。
type AgentEventWorkContext interface{ agentContext() }

func (AgentEventContext) agentContext() {}

type AgentEventSubagentContext struct {
	AgentEventContext
	// RequesterSessionKey 指向父 Agent 的会话路由键。
	RequesterSessionKey string `json:"requesterSessionKey"`
}
type AgentContextTarget struct {
	Type         string
	Account      string
	GroupID      string
	ChannelID    string
	PostID       string
	ParentPostID *string
}
type BuildAgentContextOptions struct {
	Target    AgentContextTarget
	MessageID string
}
type BuildAgentSubagentContextOptions struct {
	ParentContext AgentEventContext
	SubagentID    *string
}
type AgentEventLLMUsage struct {
	// Input、Output、CacheRead、CacheWrite 和 Total 分别记录输入、输出、缓存读取、缓存写入和总 token 数。
	Input      float64                `json:"input,omitempty"`
	Output     float64                `json:"output,omitempty"`
	CacheRead  float64                `json:"cacheRead,omitempty"`
	CacheWrite *float64               `json:"cacheWrite,omitempty"`
	Total      float64                `json:"total,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

type AgentSubagentOutcome string

const (
	AgentSubagentOK        AgentSubagentOutcome = "ok"
	AgentSubagentError     AgentSubagentOutcome = "error"
	AgentSubagentCancelled AgentSubagentOutcome = "cancelled"
)

type AgentEventLLMInputData struct {
	// Model 是使用的模型名称。
	Model string `json:"model"`
	// Prompt 是本次模型调用的输入。
	Prompt     string                 `json:"prompt"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentEventLLMOutputData struct {
	// AssistantTexts 是模型回复正文；某一轮可能仅产生工具调用而没有正文，此时使用空切片。
	AssistantTexts []string `json:"assistantTexts"`
	// Thinking 是 Agent 对用户可见的执行计划或进展说明，例如“我将查询北京天气”；不是模型 API 的隐藏 reasoning/thinking。没有内容时使用空字符串。
	Thinking string `json:"thinking"`
	// Usage 是本次模型调用的 token 用量。
	Usage      AgentEventLLMUsage     `json:"usage"`
	Extensions map[string]interface{} `json:"-"`
}
type AgentEventBeforeToolCallData struct {
	// ToolCallID 是模型输出的调用 ID，同一次调用的前后事件必须复用。
	ToolCallID string                 `json:"toolCallId"`
	ToolName   string                 `json:"toolName"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

type AgentEventAfterToolCallData struct {
	// ToolCallID 是模型输出的调用 ID，同一次调用的前后事件必须复用。
	ToolCallID string                 `json:"toolCallId"`
	ToolName   string                 `json:"toolName"`
	Result     interface{}            `json:"result"`
	Extensions map[string]interface{} `json:"-"`
}

type AgentEventSubagentSpawnedData struct {
	// AgentID 和 Label 分别是子 Agent 标识和展示名称。
	AgentID    string                 `json:"agentId,omitempty"`
	Label      string                 `json:"label,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

type AgentEventSubagentEndedData struct {
	// Outcome 和 Reason 分别是子 Agent 执行结果和结束原因。
	Outcome    AgentSubagentOutcome   `json:"outcome"`
	Reason     string                 `json:"reason,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}
