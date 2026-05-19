package tgbot

import (
	"fmt"
	"strings"

	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/consts"
	"github.com/changlongH/gantry/executor"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	cfgMgr *config.Manager
	exe    *executor.Executor
}

func NewBot(cfgMgr *config.Manager) (*Bot, error) {
	// 初始化时使用当前 Token
	api, err := tgbotapi.NewBotAPI(cfgMgr.Get().Telegram.Token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		api:    api,
		cfgMgr: cfgMgr,
		exe:    executor.NewExecutor(cfgMgr),
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			go b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Bot) SendDeploymentMenu(envName, commitHash, commitMsg, author string) error {
	cfg := b.cfgMgr.Get()
	_, ok := cfg.Envs[envName]
	if !ok {
		return fmt.Errorf("env not found")
	}

	text := fmt.Sprintf("🚀 **【Git Action】已推送到目标服务器！**\n\n"+
		"🌍 **环境:** `%s`\n"+
		"👤 **提交者:** %s\n"+
		"🏷 **提交 SHA:** `%s`\n"+
		"💬 **日志:** %s\n\n"+
		"选择容器管理操作:",
		envName, author, commitHash[:7], commitMsg)

	msg := tgbotapi.NewMessage(cfg.Telegram.ChatID, text)
	msg.ParseMode = "Markdown"

	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, svc := range cfg.Apps {
		data := fmt.Sprintf("%s|%s|%s", consts.ActionBuild, envName, svc)
		btn := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔨 Build %s", svc), data)
		keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(btn))
	}

	restartAllData := fmt.Sprintf("%s|%s", consts.ActionRestart, envName)
	restartBtn := tgbotapi.NewInlineKeyboardButtonData("🔄 Compose Up (Restart)", restartAllData)
	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(restartBtn))

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	callback := tgbotapi.NewCallback(query.ID, "Triggering automation...")
	b.api.Request(callback)

	data := strings.Split(query.Data, "|")
	if len(data) < 2 {
		return
	}

	action := data[0]
	envName := data[1]

	chatID := b.cfgMgr.Get().Telegram.ChatID
	statusMsg, _ := b.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ Remote runner: Executing `%s` on environment `%s`...", action, envName)))

	var imageTag = consts.GenImageTagByStrategy(b.cfgMgr.Get().Envs[envName].ImageTagStrategy, envName)
	var result string
	var err error

	switch consts.ActionType(action) {
	case consts.ActionBuild:
		if len(data) == 3 {
			svc := data[2]
			result, err = b.exe.BuildAndPushService(envName, svc, false, imageTag)
			if err == nil {
				b.sendPushMenu(envName, svc)
			}
		}
	case consts.ActionRestart:
		result, err = b.exe.RestartCompose(envName, false)
	}

	b.updateStatusMessage(statusMsg.MessageID, action, result, err)
}

func (b *Bot) sendPushMenu(envName, svc string) {
	text := fmt.Sprintf("✅ Build complete for service `%s`.\nDeploy registry transmission required?", svc)
	msg := tgbotapi.NewMessage(b.cfgMgr.Get().Telegram.ChatID, text)
	msg.ParseMode = "Markdown"

	pushData := fmt.Sprintf("push|%s|%s", envName, svc)
	pushBtn := tgbotapi.NewInlineKeyboardButtonData("📤 Push to Registry", pushData)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(pushBtn))
	b.api.Send(msg)
}

func (b *Bot) updateStatusMessage(msgID int, action, result string, err error) {
	var text string
	if err != nil {
		text = fmt.Sprintf("❌ Action `%s` execution anomaly:\n```\n%v\n%s\n```", action, err, result)
	} else {
		text = fmt.Sprintf("✅ Action `%s` completed successfully.\n```\n%s\n```", action, result)
	}

	if len(text) > 4000 {
		text = text[:4000] + "\n...[Output truncated due to payload constraints]"
	}

	edit := tgbotapi.NewEditMessageText(b.cfgMgr.Get().Telegram.ChatID, msgID, text)
	edit.ParseMode = "Markdown"
	b.api.Send(edit)
}
