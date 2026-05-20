package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/changlongH/gantry/config"
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
