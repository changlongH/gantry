package consts

import (
	"fmt"
	"time"
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

func GenImageTagByStrategy(strategy string, isProd bool) string {
	switch strategy {
	case ImageTagStrategyTimestamp:
		if isProd {
			return fmt.Sprintf("%s-prod", time.Now().Format("20060102-150405"))
		} else {
			return fmt.Sprintf("%s", time.Now().Format("20060102-150405"))
		}
	default:
		return ImageTagStrategyLatest
	}
}
