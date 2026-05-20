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
	// === 1. 菜单切换路由 (原地更新旧消息) ===
	case ActionMenuMain:
		b.switchMenu(ctx, bot, chatID, msgID, baseText, b.buildMainMenu(data.Env))
	case ActionMenuBuild:
		b.switchMenu(ctx, bot, chatID, msgID, baseText, b.buildServiceMenu(data.Env, ActionMenuBuild))
	case ActionMenuRestart:
		b.switchMenu(ctx, bot, chatID, msgID, baseText, b.buildServiceMenu(data.Env, ActionMenuRestart))

		// === 2. 执行动作路由 (更新消息提示执行中，异步执行具体操作，最后再更新消息显示结果) ===
	case ActionDoBuild:
		b.asyncBuildTask(ctx, bot, chatID, data)
	case ActionDoRestart:
		b.asyncRestartTask(ctx, bot, chatID, data)
	default:
		return fmt.Errorf("unknown action: %s", data.Action)
	}

	return nil
}

func (b *Bot) asyncBuildTask(ctx context.Context, bot *telego.Bot, chatID int64, data CallbackData) {
	// 发送初始状态消息
	statusMsg, err := bot.SendMessage(ctx, tu.Message(
		tu.ID(chatID),
		fmt.Sprintf("⏳ **[构建任务启动]**\n👤 **服务名称:** `%s`\n🌍 **目标环境:** `%s`\n📊 **当前状态:** 正在编译并打包...", data.Svc, data.Env),
	).WithParseMode(telego.ModeMarkdown))
	if err != nil {
		return
	}

	var envDesc = data.Env
	if envCfg, ok := b.cfgMgr.Get().Envs[data.Env]; ok {
		envDesc = envCfg.Desc
	}

	go func() {
		// 执行构建
		out, imageID, err := b.exe.BuildAndPushService(data.Env, data.Svc, false, "")

		// 组装最终文本
		var resultText string
		if err != nil {
			resultText = fmt.Sprintf("❌ **[构建任务失败]**\n👤 **服务名称:** `%s`\n🌍 **目标环境:** `%s`\n⚠️ **错误信息:** `%v`", data.Svc, envDesc, err)
		} else {
			shortID := imageID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			resultText = fmt.Sprintf("✅ **[构建任务成功]**\n👤 **服务名称:** `%s`\n🌍 **目标环境:** `%s`\n🆔 **镜像 ID:** `%s`", data.Svc, envDesc, shortID)
		}

		// 附加安全拦截后的日志输出
		resultText = b.appendLogBlock(resultText, out)

		// 更新状态消息 (使用 Background 防止上层 context 提前取消)
		editParams := tu.EditMessageText(tu.ID(chatID), statusMsg.MessageID, resultText).WithParseMode(telego.ModeMarkdown)
		_, _ = bot.EditMessageText(context.Background(), editParams)
	}()
}

func (b *Bot) asyncRestartTask(ctx context.Context, bot *telego.Bot, chatID int64, data CallbackData) {
	// 发送初始状态消息
	statusMsg, err := bot.SendMessage(ctx, tu.Message(
		tu.ID(chatID),
		fmt.Sprintf("⏳ **[重启任务启动]**\n👤 **服务名称:** `%s`\n🌍 **目标环境:** `%s`\n📊 **当前状态:** 正在重启容器群...", data.Svc, data.Env),
	).WithParseMode(telego.ModeMarkdown))
	if err != nil {
		return
	}

	go func() {
		// 执行重启
		out, err := b.exe.RestartCompose(data.Env, []string{data.Svc}, true, false)

		// 组装最终文本
		var resultText string
		var envDesc = data.Env
		if envCfg, ok := b.cfgMgr.Get().Envs[data.Env]; ok {
			envDesc = envCfg.Desc
		}
		if err != nil {
			resultText = fmt.Sprintf("❌ **[重启任务失败]**\n👤 **服务名称:** `%s`\n🌍 **目标环境:** `%s`\n⚠️ **错误信息:** `%v`", data.Svc, envDesc, err)
		} else {
			resultText = fmt.Sprintf("✅ **[重启任务成功]**\n👤 **服务名称:** `%s`\n🌍 **目标环境:** `%s`\n📊 **当前状态:** 服务已成功拉起并处于活跃状态。", data.Svc, envDesc)
		}

		// 附加安全拦截后的日志输出
		resultText = b.appendLogBlock(resultText, out)

		// 更新状态消息
		editParams := tu.EditMessageText(tu.ID(chatID), statusMsg.MessageID, resultText).WithParseMode(telego.ModeMarkdown)
		_, _ = bot.EditMessageText(context.Background(), editParams)
	}()
}

func (b *Bot) appendLogBlock(baseText, logOut string) string {
	if len(logOut) == 0 {
		return baseText
	}

	maxLogLen := 500
	if len(logOut) > maxLogLen {
		// 保留尾部核心错误日志
		logOut = logOut[len(logOut)-maxLogLen:]
		return baseText + fmt.Sprintf("\n\n📄 **控制台输出 (截取尾部):**\n```\n%s\n```", logOut)
	}

	return baseText + fmt.Sprintf("\n\n📄 **控制台输出:**\n```\n%s\n```", logOut)
}
