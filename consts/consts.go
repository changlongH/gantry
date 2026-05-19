package consts

import (
	"fmt"
	"time"
)

const (
	// 环境选项
	EnvProd = "prod"
	EnvTest = "test"
)

const (
	BranchRelease = "release"
	BranchDevelop = "develop"
	BranchMain    = "main"
)

const (
	ImageTagStrategyLatest    = "latest"    // 固定标签
	ImageTagStrategyTimestamp = "timestamp" // 时间戳格式 20260102-150405-prod
)

func GenImageTagByStrategy(strategy string, envName string) string {
	switch strategy {
	case ImageTagStrategyTimestamp:
		return fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), envName)
	default:
		return ImageTagStrategyLatest
	}
}
