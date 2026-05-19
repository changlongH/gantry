package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/consts"
)

type BuildOptions struct {
	EnvName  string
	Service  string
	ImageTag string
}

type Executor struct {
	cfgMgr *config.Manager
	mu     sync.Mutex // 互斥锁 禁止并发执行
}

func NewExecutor(cfgMgr *config.Manager) *Executor {
	return &Executor{cfgMgr: cfgMgr}
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
	buildCtx := filepath.Join(env.ProjectPath, svc)

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
		return e.runCmd(env.ProjectPath, streamOutput, "docker", "push", imageName)
	}
	return buildRet, nil
}

func (e *Executor) RestartCompose(envName string, streamOutput bool) (string, error) {
	cfg := e.cfgMgr.Get()
	env, ok := cfg.Envs[envName]
	if !ok {
		return "", fmt.Errorf("environment %s not found", envName)
	}

	dir := filepath.Dir(env.ComposeFile)
	file := filepath.Base(env.ComposeFile)

	return e.runCmd(dir, streamOutput, "docker-compose", "-f", file, "up", "-d")
}

func (e *Executor) GetSvcImageName(opts BuildOptions) string {
	cfg := e.cfgMgr.Get()
	env := cfg.Envs[opts.EnvName]

	imageTag := opts.ImageTag
	if imageTag == "" {
		imageTag = consts.GenImageTagByStrategy(env.ImageTagStrategy, opts.EnvName)
	}

	sanitizedSvc := strings.ReplaceAll(opts.Service, "/", "-")
	image := fmt.Sprintf("%s:%s", sanitizedSvc, imageTag)

	if env.Registry != "" {
		image = fmt.Sprintf("%s/%s", env.Registry, image)
	}
	return image
}

func (e *Executor) runCmd(dir string, streamOutput bool, name string, args ...string) (string, error) {
	// 尝试获取锁，如果锁被占用，立即返回错误
	if !e.mu.TryLock() {
		return "", fmt.Errorf("⚠️  系统繁忙：当前已有构建任务正在进行中，请等待结束后再操作。")
	}
	defer e.mu.Unlock()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	fmt.Printf("📢 执行命令: %s %s (工作目录: %s)\n", name, strings.Join(args, " "), dir)

	var buf bytes.Buffer
	if streamOutput {
		cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}

	err := cmd.Run()
	return buf.String(), err
}
