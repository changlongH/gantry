package executor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/changlongH/gantry/consts"
)

func (e *Executor) GetSvcImageName(opts BuildOptions) string {
	cfg := e.cfgMgr.Get()
	envCfg := cfg.Envs[opts.EnvName]

	imageTag := opts.ImageTag
	if imageTag == "" {
		imageTag = consts.GenImageTagByStrategy(envCfg.Docker.ImageTagStrategy, envCfg.IsProd)
	}

	sanitizedSvc := strings.ReplaceAll(opts.Service, "/", "-")
	image := fmt.Sprintf("%s:%s", sanitizedSvc, imageTag)

	if envCfg.Docker.Registry != "" {
		if strings.HasSuffix(envCfg.Docker.Registry, "/") {
			image = fmt.Sprintf("%s%s", envCfg.Docker.Registry, image)
		} else {
			image = fmt.Sprintf("%s/%s", envCfg.Docker.Registry, image)
		}
	}
	return image
}

// CleanupDanglingImages 清理悬空镜像（dangling images）
func (e *Executor) CleanupDanglingImages() {
	cmd := exec.Command("docker", "image", "prune", "-f")
	cmd.Run()
}

func (e *Executor) BuildAndPushService(envName, svc string, streamOutput bool, imageTag string) (string, string, error) {
	// 每次执行动态获取最新配置
	cfg := e.cfgMgr.Get()
	envCfg, ok := cfg.Envs[envName]
	if !ok {
		return "", "", fmt.Errorf("environment %s not found", envName)
	}

	opts := BuildOptions{
		EnvName:  envName,
		Service:  svc,
		ImageTag: imageTag,
	}

	imageName := e.GetSvcImageName(opts)
	buildCtx := filepath.Join(envCfg.OutputPath, svc)

	// 基础构建参数
	cmdArgs := []string{"build", "-t", imageName}

	// 拼接命令参数: --build-arg KEY=VALUE
	for k, v := range envCfg.Docker.BuildArgs {
		cmdArgs = append(cmdArgs, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	// 是否指定dockerfile
	if envCfg.Docker.Dockerfile != "" && envCfg.Docker.Dockerfile != "Dockerfile" {
		cmdArgs = append(cmdArgs, "-f", envCfg.Docker.Dockerfile)
	}

	cmdArgs = append(cmdArgs, ".")

	buildRet, err := e.runCmd(buildCtx, streamOutput, "docker", cmdArgs...)
	if err != nil {
		return buildRet, "", fmt.Errorf("build failed: %w", err)
	}

	if envCfg.Docker.Registry != "" {
		out, err := e.runCmd(buildCtx, streamOutput, "docker", "push", imageName)
		if err != nil {
			return out, imageName, fmt.Errorf("push failed: %w", err)
		}
	}
	return buildRet, imageName, nil
}

func (e *Executor) RestartCompose(envName string, services []string, force bool, streamOutput bool) (string, error) {
	cfg := e.cfgMgr.Get()
	envCfg, ok := cfg.Envs[envName]
	if !ok {
		return "", fmt.Errorf("environment %s not found", envName)
	}

	dir := filepath.Dir(envCfg.Docker.ComposeFile)
	file := filepath.Base(envCfg.Docker.ComposeFile)

	// 基础命令
	cmdArgs := []string{"-f", file, "up", "-d"}

	// 如果设置了强制重启，添加参数
	if force {
		cmdArgs = append(cmdArgs, "--force-recreate")
	}

	// 如果指定了服务列表，则只处理这些服务
	if len(services) > 0 {
		cmdArgs = append(cmdArgs, "--no-deps") // 可选：不自动拉起依赖容器，仅重启目标
		cmdArgs = append(cmdArgs, services...)
	}

	return e.runCmd(dir, streamOutput, "docker", append([]string{"compose"}, cmdArgs...)...)
}
