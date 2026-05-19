package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/consts"
	"github.com/changlongH/gantry/executor"
)

type InteractiveCLI struct {
	cfgMgr *config.Manager
	exe    *executor.Executor
}

func NewInteractiveCLI(cfgMgr *config.Manager) *InteractiveCLI {
	return &InteractiveCLI{
		cfgMgr: cfgMgr,
		exe:    executor.NewExecutor(cfgMgr),
	}
}

func (cli *InteractiveCLI) Run() {
	fmt.Println("=== 🤖 Welcome to Deploy-Bot Interactive CLI ===")

	var defaultEnvIndex int
	cfg := cli.cfgMgr.Get()
	var envs []string
	for k := range cfg.Envs {
		envs = append(envs, k)
		if k == consts.EnvTest {
			defaultEnvIndex = len(envs) - 1
		}
	}

	var selectedEnv string
	envPrompt := &survey.Select{
		Message: "🚧 请选择要操作的环境:",
		Options: envs,
		Default: defaultEnvIndex,
		Description: func(value string, index int) string {
			if envCfg, ok := cfg.Envs[value]; ok {
				return fmt.Sprintf("(%s)", envCfg.Desc)
			}
			return ""
		},
	}
	if err := survey.AskOne(envPrompt, &selectedEnv); err != nil {
		log.Fatalf("Selection interrupted: %v", err)
	}

	// 重新 Get 一次确保获取最新
	envCfg := cli.cfgMgr.Get().Envs[selectedEnv]

	if selectedEnv == consts.EnvProd {
		confirm := false
		prompt := &survey.Confirm{
			Message: "️‼️ 【重要】: 当前选择的是【生产环境】是否继续?",
			Default: false,
		}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			fmt.Println("❌ 操作已取消.")
			return
		}
	}

	var action string
	actionPrompt := &survey.Select{
		Message: fmt.Sprintf("当前环境 🚧 %s 👷 请选择:", envCfg.Desc),
		Options: []string{"🔨 构建微服务", "🔄 通过 Docker Compose 重启环境", "🚪 退出"},
	}
	survey.AskOne(actionPrompt, &action)

	switch action {
	case "🔨 构建微服务":
		cli.handleServiceBuild(selectedEnv, envCfg)
	case "🔄 通过 Docker Compose 重启环境":
		cli.handleComposeRestart(selectedEnv)
	default:
		fmt.Println("再见!")
		os.Exit(0)
	}
}

func (cli *InteractiveCLI) handleServiceBuild(envName string, envCfg config.Environment) {
	// 支持多个服务构建，用户可以选择一个或多个服务进行构建
	var selectedSvcs []string

	customIcons := func(icons *survey.IconSet) {
		//icons.SelectFocus.Format = ""     // 选中焦点时的箭头
		icons.Question.Format = "❓"       // 问题图标
		icons.MarkedOption.Text = "[✅]"   // 选中的勾选框
		icons.UnmarkedOption.Text = "[x]" // 未选中的框
	}

	svcPrompt := &survey.MultiSelect{
		Message: "选择要构建的服务 (使用 [空格键] 勾选/取消，[回车] 确认):",
		Options: cli.cfgMgr.Get().Apps,
	}
	if err := survey.AskOne(svcPrompt, &selectedSvcs, survey.WithIcons(customIcons)); err != nil {
		return
	}

	// 如果用户什么都没选直接回车
	if len(selectedSvcs) == 0 {
		fmt.Println("⚠️ 未选择任何服务，操作已取消。")
		return
	}

	var imageTag = consts.GenImageTagByStrategy(envCfg.ImageTagStrategy, envName)
	// 用于记录批量构建中，哪些服务真正构建成功了（后续只推送成功的服务）
	var successSvcs []string

	// 循环遍历执行构建 + 推送
	for _, selectedSvc := range selectedSvcs {
		fmt.Printf("\n🚀 开始构建 [%s] [%s] ...\n", envCfg.Desc, selectedSvc)
		_, err := cli.exe.BuildAndPushService(envName, selectedSvc, true, imageTag)
		if err != nil {
			// 行业标准规范：批量构建时，某个微服务失败不应该直接 return 阻断流程，应当记录并继续构建下一个
			fmt.Printf("❌ [%s] [%s] 构建失败: %v\n", envCfg.Desc, selectedSvc, err)
			continue
		}
		fmt.Printf("✅ [%s] [%s] 构建成功!\n", envCfg.Desc, selectedSvc)
		successSvcs = append(successSvcs, selectedSvc)
	}

	// 如果全部都构建失败，则直接返回
	if len(successSvcs) == 0 {
		fmt.Println("\n❌ 所有选中的服务均构建失败。")
		return
	}
}

func (cli *InteractiveCLI) handleComposeRestart(envName string) {
	fmt.Printf("\n🔄 通过 Docker Compose 重启 [%s] 环境...\n", envName)
	_, err := cli.exe.RestartCompose(envName, true)
	if err != nil {
		fmt.Printf("\n❌ 重启失败: %v\n", err)
		return
	}
	fmt.Println("\n✅ Docker-compose 服务已启动并运行!")
}
