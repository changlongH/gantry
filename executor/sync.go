package executor

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SyncCode 同步代码到目标服务器 (支持多步骤差异化同步)
func (e *Executor) SyncCode(envName string, streamOutput bool) (string, error) {
	cfg := e.cfgMgr.Get()
	env, ok := cfg.Envs[envName]
	if !ok {
		return "", fmt.Errorf("environment %s not found", envName)
	}

	syncCfg := env.Sync
	// 基础合法性校验
	if len(syncCfg.RemoteIPs) == 0 {
		return "", fmt.Errorf("no remote IPs configured for environment %s", envName)
	}
	if len(syncCfg.Steps) == 0 {
		return "", fmt.Errorf("no sync steps configured for environment %s", envName)
	}

	// 假设本地源码根路径存在 env.SrcPath 中，如果为空则需要你根据实际情况指定
	localBaseCtx := env.SrcPath
	if localBaseCtx == "" {
		return "", fmt.Errorf("local source path (SrcPath) is not configured for environment %s", envName)
	}

	var totalOutput strings.Builder

	// 外层循环：遍历所有线上服务器 IP
	for _, ip := range syncCfg.RemoteIPs {
		heading := fmt.Sprintf("\n========== 开始同步服务器: [%s] ==========\n", ip)
		if streamOutput {
			fmt.Print(heading)
		}
		totalOutput.WriteString(heading)

		// 内层循环：按顺序执行 YAML 中配置的每一个步骤
		for idx, step := range syncCfg.Steps {
			stepHeading := fmt.Sprintf("--> 步骤 [%d/%d]: %s\n", idx+1, len(syncCfg.Steps), step.Name)
			if streamOutput {
				fmt.Print(stepHeading)
			}
			totalOutput.WriteString(stepHeading)

			// 1. 动态计算本地绝对/相对路径
			// filepath.Join 会自动处理空字符串，并且规范化路径
			localPath := filepath.Join(localBaseCtx, step.LocalSubPath)
			// rsync 的黄金铁律：本地目录结尾必须带斜杠，否则会传输整个目录本身
			if !strings.HasSuffix(localPath, "/") {
				localPath += "/"
			}

			// 2. 动态计算远程目标路径
			remotePath := syncCfg.RemotePath
			if step.RemoteSubPath != "" {
				remotePath = filepath.Join(remotePath, step.RemoteSubPath)
			}
			remoteAddr := fmt.Sprintf("%s@%s:%s", syncCfg.RemoteUser, ip, remotePath)

			// 3. 解析并组装 rsync 基础参数 (例如将 "-avz --delete" 切分为 slice)
			args := strings.Fields(step.RsyncOptions)

			// 4. 注入指定的 SSH 密钥参数 (强加 StrictHostKeyChecking 防止首次连接卡住)
			sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", syncCfg.RsyncKey)
			args = append(args, "-e", sshCmd)

			// 5. 注入默认排除规则与用户自定义排除规则
			args = append(args, "--exclude=.git") // 默认安全策略：不带走 git 历史
			for _, ex := range step.Exclude {
				if ex != "" {
					args = append(args, fmt.Sprintf("--exclude=%s", ex))
				}
			}

			// 6. 追加源路径和目标路径
			args = append(args, localPath, remoteAddr)

			// 打印当前步骤实际拼接出来的命令，方便排查
			cmdPrompt := fmt.Sprintf("执行命令: rsync %s\n", strings.Join(args, " "))
			if streamOutput {
				fmt.Print(cmdPrompt)
			}
			totalOutput.WriteString(cmdPrompt)

			// 7. 调用原本已有的 runCmd 驱动底层的 exec.Command
			// 注意：runCmd 内部通常会直接把日志实时打印到控制台
			out, err := e.runCmd(localBaseCtx, streamOutput, "rsync", args...)
			totalOutput.WriteString(out)
			if err != nil {
				errResult := fmt.Sprintf("❌ 步骤 [%s] 同步失败: %v\n", step.Name, err)
				totalOutput.WriteString(errResult)
				return totalOutput.String(), fmt.Errorf("服务器 %s 步骤 [%s] 失败: %w", ip, step.Name, err)
			}

			successResult := fmt.Sprintf("✓ 步骤 [%s] 同步成功\n\n", step.Name)
			totalOutput.WriteString(successResult)
		}
	}

	finalMsg := "\n🎉 所有服务器及步骤代码同步全部完成！\n"
	if streamOutput {
		fmt.Print(finalMsg)
	}
	totalOutput.WriteString(finalMsg)

	return totalOutput.String(), nil
}
