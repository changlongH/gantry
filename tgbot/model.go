package tgbot

// 建议定义在 tgbot/types.go
type CallbackData struct {
	Action string `json:"a"`
	Env    string `json:"e"`
	Svc    string `json:"s,omitempty"`
}
