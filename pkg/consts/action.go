package consts

type ActionType string

const (
	ActionBuild ActionType = "build" // 构建服务 + 提交镜像
	//ActionPush    ActionType = "push"    // 推送镜像
	ActionRestart ActionType = "restart" // 重启服务
)
