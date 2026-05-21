package tgbot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type CallbackAction string

const (
	ActionMenuMain    CallbackAction = "menu_main"    // 返回主菜单
	ActionMenuBuild   CallbackAction = "menu_build"   // 显示构建服务的子菜单
	ActionMenuRestart CallbackAction = "menu_restart" // 显示重启服务的子菜单

	ActionDoBuild   CallbackAction = "do_build"   // 执行构建
	ActionDoRestart CallbackAction = "do_restart" // 执行重启
)

type CallbackData struct {
	Action string `json:"a"`
	Env    string `json:"e"`
	Svc    string `json:"s,omitempty"`
}

func genCallbackData(action CallbackAction, env, svc string) string {
	data, _ := json.Marshal(CallbackData{Action: string(action), Env: env, Svc: svc})
	return string(data)
}

// buildMainMenu 构造第一级主菜单
func (b *Bot) buildMainMenu(env string) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🔨 构建服务").WithCallbackData(genCallbackData(ActionMenuBuild, env, "")),
			tu.InlineKeyboardButton("🔄 重启服务").WithCallbackData(genCallbackData(ActionMenuRestart, env, "")),
		),
		/*
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("🚀 同步本地代码").WithCallbackData(genCallbackData(consts.ActionSync, env, "")),
			),
		*/
	)
}

// buildServiceMenu 构造第二级子菜单（选择要构建的具体服务）
func (b *Bot) buildServiceMenu(env string, subMenu CallbackAction) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton

	var colBtnCount = 3 // 每行按钮数量

	switch subMenu {
	case ActionMenuBuild:
		var row []telego.InlineKeyboardButton
		for i, app := range b.cfgMgr.GetAppServices(env) {
			if i%colBtnCount == 0 && i != 0 {
				rows = append(rows, row)
				row = []telego.InlineKeyboardButton{}
			}
			btn := tu.InlineKeyboardButton(fmt.Sprintf("🔨 %s", app)).
				WithCallbackData(genCallbackData(ActionDoBuild, env, app))
			row = append(row, btn)
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	case ActionMenuRestart:
		var row []telego.InlineKeyboardButton
		for i, app := range b.cfgMgr.GetDockerComposeServices(env) {
			if i%colBtnCount == 0 && i != 0 {
				rows = append(rows, row)
				row = []telego.InlineKeyboardButton{}
			}
			btn := tu.InlineKeyboardButton(fmt.Sprintf("🔄 %s", app)).
				WithCallbackData(genCallbackData(ActionDoRestart, env, app))
			row = append(row, btn)
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	// 增加“返回上一级”按钮
	backBtn := tu.InlineKeyboardButton("🔙 返回上一级").
		WithCallbackData(genCallbackData("menu_main", env, ""))
	rows = append(rows, tu.InlineKeyboardRow(backBtn))

	return tu.InlineKeyboard(rows...)
}

func (b *Bot) switchMenu(ctx context.Context, bot *telego.Bot, chatID int64, msgID int, text string, menu *telego.InlineKeyboardMarkup) {
	editMsg := tu.EditMessageText(tu.ID(chatID), msgID, text).WithReplyMarkup(menu)
	_, _ = bot.EditMessageText(ctx, editMsg)
}

// SendDeploymentMenu 接收代码更新通知，并渲染主操作菜单
func (b *Bot) SendDeploymentMenu(ctx context.Context, envName, commitHash, commitMsg, author string) error {
	cfg := b.cfgMgr.Get()
	envCfg, ok := cfg.Envs[envName]
	if !ok {
		return fmt.Errorf("环境配置不存在: %s", envName)
	}

	text := fmt.Sprintf("🚀 **【代码更新通知】**\n"+
		"🌍 **当前环境:** `%s`\n"+
		"👤 **提交人员:** %s\n"+
		"🏷 **Commit:** `%s`\n"+
		"💬 **提交日志:** %s\n\n"+
		"👇 **请选择操作:**",
		envCfg.Desc, author, commitHash[:7], commitMsg)

	msg := tu.Message(tu.ID(cfg.Telegram.ChatID), text).
		WithParseMode(telego.ModeMarkdown).
		WithReplyMarkup(b.buildMainMenu(envName))

	_, err := b.bot.SendMessage(ctx, msg)
	return err
}
