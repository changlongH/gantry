package executor

import "fmt"

// SyncCode 同步代码到目标服务器
func (e *Executor) SyncCode(envName string, streamOutput bool) (string, error) {
	cfg := e.cfgMgr.Get()
	env, ok := cfg.Envs[envName]
	if !ok {
		return "", fmt.Errorf("environment %s not found", envName)
	}

	var localPath string
	var remoteHost string
	var remoteUser string
	var remotePath string
	var remoteAddr = fmt.Sprintf("%s@%s:%s", remoteUser, remoteHost, remotePath)

	// 使用 rsync 进行增量同步
	// rsync -avz --exclude='.git' /本地路径/ root@目标IP:/服务器路径/
	args := []string{"-avz", "--delete", "--exclude=.git", localPath + "/", remoteAddr}

	fmt.Printf("SyncCode [%s] from [%s] to [%s]\n", env.Desc, localPath, remoteAddr)

	return e.runCmd(localPath, streamOutput, "rsync", args...)
}
