package cmd

import (
	"os"

	"github.com/changlongH/gantry/pkg/config"
)

func (cli *InteractiveCLI) buildMainMenu(envCfg config.Environment) []MenuAction {
	var actions = []MenuAction{}
	if envCfg.Docker != nil {
		actions = append(actions,
			MenuAction{Label: "🔨 构建服务 (Build)", Handler: cli.handleServiceBuild},
			MenuAction{Label: "🔄 重启容器 (Compose)", Handler: cli.handleComposeRestart},
			MenuAction{Label: "🧹 清理镜像 (Prune)", Handler: cli.handlePruneImages},
		)
	}
	if envCfg.Sync != nil {
		actions = append(actions, MenuAction{Label: "📦 同步源码 (Sync)", Handler: cli.handleSyncSource})
	}
	actions = append(actions, MenuAction{Label: "🚪 退出程序 (Exit)", Handler: func(e string, c config.Environment) { os.Exit(0) }})
	return actions
}
