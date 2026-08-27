package model

// Message 对应前端 Dialog.vue 的消息结构
type Message struct {
	Type     string `json:"type"`               // "user" | "server"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	FileURL  string `json:"fileUrl,omitempty"`
	Meta     *Meta  `json:"meta,omitempty"`
}

type Meta struct {
	ServerStatusHint int `json:"serverStatusHint"`
}

// NewServerMessage 构造服务端回复。hint: 0=放行输入，1=禁用输入（忙）。
func NewServerMessage(text string, hint int) Message {
	return Message{Type: "server", Text: text, Meta: &Meta{ServerStatusHint: hint}}
}
