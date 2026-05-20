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
			tu.InlineKeyboardButton("🔨 构建微服务").WithCallbackData(genCallbackData(ActionMenuBuild, env, "")),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🔄 强制重启 (Compose Up)").WithCallbackData(genCallbackData(ActionMenuRestart, env, "")),
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
	cfg := b.cfgMgr.Get()

	switch subMenu {
	case ActionMenuBuild:
		// 遍历应用列表生成构建按钮
		for _, app := range cfg.Apps {
			btn := tu.InlineKeyboardButton(fmt.Sprintf("🔨 %s", app)).
				WithCallbackData(genCallbackData(ActionDoBuild, env, app))
			rows = append(rows, tu.InlineKeyboardRow(btn))
		}
	case ActionMenuRestart:
		// 遍历应用列表生成重启按钮
		for _, app := range cfg.Apps {
			btn := tu.InlineKeyboardButton(fmt.Sprintf("🔄 %s", app)).
				WithCallbackData(genCallbackData(ActionDoRestart, env, app))
			rows = append(rows, tu.InlineKeyboardRow(btn))
		}
	}

	// 增加“返回上一级”按钮
	backBtn := tu.InlineKeyboardButton("🔙 返回上一级").
		WithCallbackData(genCallbackData("menu_main", env, ""))
	rows = append(rows, tu.InlineKeyboardRow(backBtn))

	return tu.InlineKeyboard(rows...)
}

// SendDeploymentMenu 接收代码更新通知，并渲染主操作菜单
func (b *Bot) SendDeploymentMenu(ctx context.Context, envName, commitHash, commitMsg, author string) error {
	cfg := b.cfgMgr.Get()
	if _, ok := cfg.Envs[envName]; !ok {
		return fmt.Errorf("环境配置不存在: %s", envName)
	}

	text := fmt.Sprintf("🚀 **【代码推送通知】**\n\n"+
		"🌍 **目标环境:** `%s`\n"+
		"👤 **提交人员:** %s\n"+
		"🏷 **Commit:** `%s`\n"+
		"💬 **更新日志:** %s\n\n"+
		"👇 **请选择运维操作:**",
		envName, author, commitHash[:7], commitMsg)

	msg := tu.Message(tu.ID(cfg.Telegram.ChatID), text).
		WithParseMode(telego.ModeMarkdown).
		WithReplyMarkup(b.buildMainMenu(envName))

	_, err := b.bot.SendMessage(ctx, msg)
	return err
}
