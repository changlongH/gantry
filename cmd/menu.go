package cmd

import (
	"os"

	"github.com/changlongH/gantry/config"
)

func (cli *InteractiveCLI) buildMainMenu(envCfg config.Environment) []MenuAction {
	var actions = []MenuAction{
		{Label: "🔨 构建服务 (Build)", Handler: cli.handleServiceBuild},
	}
	if envCfg.Sync != nil {
		return []MenuAction{
			{Label: "📦 同步源码 (Sync)", Handler: cli.handleSyncSource},
		}
	}
	if envCfg.Docker != nil {
		return []MenuAction{
			{Label: "🔄 重启容器 (Compose)", Handler: cli.handleComposeRestart},
			{Label: "🧹 清理镜像 (Prune)", Handler: cli.handlePruneImages},
		}
	}
	if envCfg.Sync != nil {
		return []MenuAction{
			{Label: "📦 同步源码 (Sync)", Handler: cli.handleSyncSource},
		}
	}

	actions = append(actions, MenuAction{Label: "🚪 退出程序 (Exit)", Handler: func(e string, c config.Environment) { os.Exit(0) }})
	return actions
}
