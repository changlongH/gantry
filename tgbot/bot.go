package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/executor"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

type Bot struct {
	bot    *telego.Bot
	cfgMgr *config.Manager
	exe    *executor.Executor
}

func NewBot(cfgMgr *config.Manager) (*Bot, error) {
	bot, err := telego.NewBot(cfgMgr.Get().Telegram.Token)
	if err != nil {
		return nil, err
	}
	return &Bot{bot: bot, cfgMgr: cfgMgr, exe: executor.NewExecutor(cfgMgr)}, nil
}

// Start 启动处理器
func (b *Bot) Start() {
	ctx := context.Background()

	updates, err := b.bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get updates: %v", err)
	}

	bh, err := th.NewBotHandler(b.bot, updates)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	bh.HandleCallbackQuery(b.handleCallback)

	defer bh.Stop()
	bh.Start()
}

func (b *Bot) handleCallback(thCtx *th.Context, query telego.CallbackQuery) error {
	// 应答回调，防止按钮转圈
	err := thCtx.Bot().AnswerCallbackQuery(thCtx.Context(), tu.CallbackQuery(query.ID))
	if err != nil {
		return err
	}

	// 解析按钮数据
	var data CallbackData
	if err := json.Unmarshal([]byte(query.Data), &data); err != nil {
		return err
	}

	ctx := thCtx.Context()
	bot := thCtx.Bot()
	chatID := query.Message.GetChat().ID
	msgID := query.Message.GetMessageID()
	// 保持头部消息不变
	baseText := query.Message.Message().Text

	switch CallbackAction(data.Action) {
	case ActionMenuMain:
		// 返回主菜单
		menu := b.buildMainMenu(data.Env)
		editMsg := tu.EditMessageText(tu.ID(chatID), msgID, baseText).WithReplyMarkup(menu)
		_, _ = bot.EditMessageText(ctx, editMsg)
	case ActionMenuBuild:
		menu := b.buildServiceMenu(data.Env, ActionMenuBuild)
		editMsg := tu.EditMessageText(tu.ID(chatID), msgID, baseText).WithReplyMarkup(menu)
		_, _ = bot.EditMessageText(ctx, editMsg)
	case ActionMenuRestart:
		menu := b.buildServiceMenu(data.Env, ActionMenuRestart)
		editMsg := tu.EditMessageText(tu.ID(chatID), msgID, baseText).WithReplyMarkup(menu)
		_, _ = bot.EditMessageText(ctx, editMsg)
	case ActionDoBuild:
		// 移除按钮，提示执行中
		_, _ = bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(chatID), msgID,
			fmt.Sprintf("⏳ 正在构建服务 `%s` [%s]...\n\n%s", data.Svc, data.Env, baseText)).
			WithParseMode(telego.ModeMarkdown))

		// 异步执行，防止阻塞其他指令
		go func() {
			out, err := b.exe.BuildAndPushService(data.Env, data.Svc, false, "")
			b.sendExecutionResult(bot, chatID, "构建", data.Svc, out, err)
		}()
	case ActionDoRestart:
		// 移除按钮，提示执行中
		_, _ = bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(chatID), msgID,
			fmt.Sprintf("⏳ 正在重启服务 `%s` [%s]...\n\n%s", data.Svc, data.Env, baseText)).
			WithParseMode(telego.ModeMarkdown))

		// 异步执行，防止阻塞其他指令
		go func() {
			out, err := b.exe.RestartCompose(data.Env, []string{data.Svc}, true, false)
			b.sendExecutionResult(bot, chatID, "重启", data.Svc, out, err)
		}()
	}

	return nil
}

// sendExecutionResult 统一格式化输出执行结果
func (b *Bot) sendExecutionResult(bot *telego.Bot, chatID int64, action, target, out string, err error) {
	var text string
	if err != nil {
		text = fmt.Sprintf("❌ %s `%s` 失败:\n```\n%v\n%s\n```", action, target, err, out)
	} else {
		text = fmt.Sprintf("✅ %s `%s` 成功!", action, target)
	}

	// 截断防止超长
	if len(text) > 4000 {
		text = text[:4000] + "\n...[输出过长被截断]"
	}

	msg := tu.Message(tu.ID(chatID), text).WithParseMode(telego.ModeMarkdown)
	_, _ = bot.SendMessage(context.Background(), msg)
}
