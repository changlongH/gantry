package executor

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// CleanupDanglingImages 清理悬空镜像（dangling images）
func (e *Executor) CleanupDanglingImages() {
	cmd := exec.Command("docker", "image", "prune", "-f")
	cmd.Run()
}

func (e *Executor) BuildAndPushService(envName, svc string, streamOutput bool, imageTag string) (string, error) {
	// 每次执行动态获取最新配置
	cfg := e.cfgMgr.Get()
	env, ok := cfg.Envs[envName]
	if !ok {
		return "", fmt.Errorf("environment %s not found", envName)
	}

	opts := BuildOptions{
		EnvName:  envName,
		Service:  svc,
		ImageTag: imageTag,
	}

	imageName := e.GetSvcImageName(opts)
	buildCtx := filepath.Join(env.BuildPath, svc)

	// 基础构建参数
	cmdArgs := []string{"build", "-t", imageName}

	// 拼接命令参数: --build-arg KEY=VALUE
	for k, v := range env.BuildArgs {
		cmdArgs = append(cmdArgs, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	// 是否指定dockerfile
	if env.Dockerfile != "" && env.Dockerfile != "Dockerfile" {
		cmdArgs = append(cmdArgs, "-f", env.Dockerfile)
	}

	cmdArgs = append(cmdArgs, ".")

	buildRet, err := e.runCmd(buildCtx, streamOutput, "docker", cmdArgs...)
	if err != nil {
		return buildRet, fmt.Errorf("build failed: %w", err)
	}

	if env.Registry != "" {
		return e.runCmd(buildCtx, streamOutput, "docker", "push", imageName)
	}
	return buildRet, nil
}
