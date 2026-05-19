package tgbot

import (
	"context"
	"encoding/json"
	"log"

	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/executor"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
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

func (b *Bot) handleCallback(ctx *th.Context, query telego.CallbackQuery) error {
	// 应答回调，防止按钮转圈
	err := ctx.Bot().AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})
	if err != nil {
		return err
	}

	// 解析数据 (假设你使用 JSON 格式)
	var data CallbackData
	if err := json.Unmarshal([]byte(query.Data), &data); err != nil {
		return err
	}

	// 业务逻辑处理
	switch data.Action {
	case "build":
		// 这里可以直接使用 ctx 进行后续的消息编辑
		// 例如：ctx.EditMessageText(...)
		go b.runBuild(ctx, data.Env, data.Svc, query.Message.GetChat().ID)
	case "restart":
		//go b.runRestart(ctx, data.Env, query.Message)
	}

	return nil
}

// 统一的执行逻辑，保持与 CLI 风格一致
func (b *Bot) runBuild(ctx *th.Context, env, svc string, chatID int64) {
	/*
		msg, _ := b.bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   fmt.Sprintf("🔨 构建 %s [%s]...", svc, env),
		})

		tag := "latest" // 逻辑同前文
		out, err := b.exe.BuildAndPushService(env, svc, false, tag)
		if err != nil {
			// 构建失败，编辑消息显示错误
			ctx.EditMessageText(fmt.Sprintf("❌ 构建失败: %v\n日志输出:\n%s", err, out))
			return
		}

		// 编辑消息显示结果
		//b.updateStatus(chatID, msg.MessageID, err, out)
	*/
}
