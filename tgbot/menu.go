package tgbot

import (
	"encoding/json"
	"fmt"

	"github.com/mymmrac/telego"
)

func (b *Bot) buildAppMenu(env string) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton

	// 生成 App 列表按钮
	for _, app := range b.cfgMgr.Get().Apps {
		data, _ := json.Marshal(CallbackData{Action: "build", Env: env, Svc: app})
		rows = append(rows, []telego.InlineKeyboardButton{
			{Text: fmt.Sprintf("🔨 %s", app), CallbackData: string(data)},
		})
	}
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}
