package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/consts"
	"github.com/changlongH/gantry/executor"
	"github.com/fatih/color"
)

type InteractiveCLI struct {
	cfgMgr *config.Manager
	exe    *executor.Executor
}

type ActionHandler func(envName string, envCfg config.Environment)

type MenuAction struct {
	Label   string
	Handler ActionHandler
}

var customIcons = func(icons *survey.IconSet) {
	//icons.SelectFocus.Format = ""     // 选中焦点时的箭头
	icons.Question.Format = "❓"       // 问题图标
	icons.MarkedOption.Text = "[✅]"   // 选中的勾选框
	icons.UnmarkedOption.Text = "[x]" // 未选中的框
}

func NewInteractiveCLI(cfgMgr *config.Manager) *InteractiveCLI {
	return &InteractiveCLI{
		cfgMgr: cfgMgr,
		exe:    executor.NewExecutor(cfgMgr),
	}
}
func (cli *InteractiveCLI) Run() {
	fmt.Println("=== 🤖 Deploy-Bot 自动化运维控制台 ===")

	// 1. 环境选择
	cfg := cli.cfgMgr.Get()
	selectedEnv := cli.selectEnvironment(cfg)
	envCfg := cli.cfgMgr.Get().Envs[selectedEnv]

	// 2. 生产环境安全拦截
	if selectedEnv == consts.EnvProd && !cli.confirmEnv(envCfg.Desc) {
		return
	}

	// 3. 定义可扩展的操作菜单
	actions := []MenuAction{
		{Label: "🔨 构建服务 (Build)", Handler: cli.handleServiceBuild},
		{Label: "🔄 重启容器 (Compose)", Handler: cli.handleComposeRestart},
		{Label: "🧹 清理镜像 (Prune)", Handler: cli.handlePruneImages},
		{Label: "🚪 退出程序 (Exit)", Handler: func(e string, c config.Environment) { os.Exit(0) }},
	}

	// 提取 Label 用于渲染
	var labels []string
	for _, a := range actions {
		labels = append(labels, a.Label)
	}

	// 4. 渲染菜单
	var selectedLabel string
	prompt := &survey.Select{
		Message: fmt.Sprintf("当前环境: %s [%s] - 请选择操作:", envCfg.Desc, selectedEnv),
		Options: labels,
	}
	survey.AskOne(prompt, &selectedLabel)

	// 5. 查找并执行对应的 Handler
	for _, a := range actions {
		if a.Label == selectedLabel {
			a.Handler(selectedEnv, envCfg)
			break
		}
	}
}

func (cli *InteractiveCLI) selectEnvironment(cfg *config.Config) string {
	var envs []string
	for k := range cfg.Envs {
		envs = append(envs, k)
	}
	// 默认选项是第一个环境，通常是测试环境
	sort.Strings(envs)
	for i, env := range envs {
		if env == consts.EnvTest {
			if i != 0 {
				// 将测试环境放到首位
				envs[0], envs[i] = envs[i], envs[0]
			}
			break
		}
	}

	var selected string
	prompt := &survey.Select{
		Message:     "🚧 请选择执行环境:",
		Options:     envs,
		Description: func(v string, i int) string { return cfg.Envs[v].Desc },
		Default:     envs[0],
	}
	survey.AskOne(prompt, &selected, survey.WithIcons(customIcons))
	return selected
}

func (cli *InteractiveCLI) confirmEnv(envDesc string) bool {
	var confirm bool
	envDescHighlighted := color.New(color.FgYellow, color.Bold).Sprintf("【%s】", envDesc)
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("‼️ 当前选择的是 %s 环境，是否继续？", envDescHighlighted),
		Default: false,
	}
	survey.AskOne(prompt, &confirm)
	return confirm
}

func (cli *InteractiveCLI) handleServiceBuild(envName string, envCfg config.Environment) {
	// 支持多个服务构建，用户可以选择一个或多个服务进行构建
	var selectedSvcs []string

	svcPrompt := &survey.MultiSelect{
		Message: "选择要构建的服务 (使用 [空格键] 勾选/取消，[回车] 确认):",
		Options: cli.cfgMgr.GetAppServices(envName),
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
		_, imageID, err := cli.exe.BuildAndPushService(envName, selectedSvc, true, imageTag)
		if err != nil {
			fmt.Printf("❌ 构建失败 [%s] [%s] %v\n", envCfg.Desc, selectedSvc, err)
			continue
		}
		fmt.Printf("✅ 构建成功 [%s] [%s] ID: %s\n", envCfg.Desc, selectedSvc, imageID)
		successSvcs = append(successSvcs, selectedSvc)
	}

	// 如果全部都构建失败，则直接返回
	if len(successSvcs) == 0 {
		fmt.Println("\n❌ 所有选中的服务均构建失败。")
		return
	}
}

func (cli *InteractiveCLI) handleComposeRestart(envName string, envCfg config.Environment) {
	// 获取该环境下的所有服务列表
	allServices := cli.cfgMgr.GetDockerComposeServices(envName)
	if len(allServices) == 0 {
		fmt.Println("⚠️ 未在 docker-compose.yaml 中找到任何有效服务。")
		return
	}

	allServices = append([]string{"全部服务"}, allServices...) // 增加一个选项用于重启全部服务
	var selectedSvcs []string
	svcPrompt := &survey.MultiSelect{
		Message: "选择要强制重启的服务（可多选）:",
		Options: allServices,
		//Default: allServices[0], // 默认选中“全部服务”
	}
	survey.AskOne(svcPrompt, &selectedSvcs, survey.WithIcons(customIcons))

	if len(selectedSvcs) <= 0 {
		fmt.Printf("👋 未选择任何服务，操作已取消。\n")
		return
	}

	// 2. 询问是否强制重建 (force)
	//force := false
	//survey.AskOne(&survey.Confirm{Message: "是否执行 --force-recreate (重建容器)?", Default: true}, &force)

	// 3. 执行重启
	fmt.Printf("⏳ 正在重启服务 %v ...\n", selectedSvcs)
	output, err := cli.exe.RestartCompose(envName, selectedSvcs, true, true)
	if err != nil {
		fmt.Printf("❌ 重启失败: %v\n%s\n", err, output)
		return
	}
	fmt.Println("✅ 指定服务已按要求重启。")
}

func (cli *InteractiveCLI) handlePruneImages(envName string, envCfg config.Environment) {
	cli.exe.CleanupDanglingImages()
}
