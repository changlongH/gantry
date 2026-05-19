# Gantry 🤖

`Gantry` 是一个基于 Go 语言开发的轻量级、跨平台自动化运维与部署工具。
它支持**本地交互式 CLI 菜单** 与 **后台 Webhook 守护进程（结合 Telegram Bot）** 双模式运行。
旨在简化多微服务环境下的 Docker 构建与 Docker Compose 容器编排工作流。

---

## 🌟 核心特性

- **双模式切换**：
  - **CLI 模式**：运维人员在服务器终端通过方向键上下选择环境、服务并执行构建与重启，支持生产环境二次确认。
  - **Server 模式**：作为后台守护进程，接收来自 GitHub Actions 等 VCS 的 Webhook 触发，通过 Telegram 交互式按钮远程控制部署。
- **动态配置热更新**：集成 Viper 与 FSNotify，无需重启服务即可平滑修改并生效服务器路径、微服务列表及私有镜像仓库等配置。
- **安全的并发控制**：内部采用读写锁（RWMutex）保护配置快照，在高并发的 Webhook 回调中保证内存安全。
- **实时日志流**：CLI 模式下实时将 Docker 构建日志输出到控制台；Server 模式下自动捕获异常并将精简日志回传至 Telegram。

---

## 层次目录结构

```text
gantry/
├── main.go             # 程序主入口
├── config.yaml         # 核心配置文件
├── go.mod              # 依赖管理
├── cmd/                # 命令行生命周期管理
│   ├── root.go         # 模式分发与引导
│   └── interactive.go  # CLI 交互式菜单逻辑
├── config/             # 配置解析与热更新管理器
│   └── config.go       
├── executor/           # 底层 Shell 命令执行器
│   └── executor.go     
├── server/             # Webhook HTTP 接收服务
│   └── server.go       
└── tgbot/              # Telegram Bot 回调与交互逻辑
    └── bot.go
```

## 🛠️ 快速开始
- 下载依赖

`go mod tidy`

- 编译为可执行文件

`go build -o gantry main.go`

## 🚀 运行指南

**Mode 1: 本地交互式 CLI 模式（默认）**

适合运维人员直接登录服务器进行手动发布或回滚。支持方向键上下选择，高亮视觉反馈。

`./gantry --mode cli --config ./config.yaml`

注：若选择 prod 环境，系统会自动弹出二次确认提示，防止误操作。

**Mode 2: Server 自动化模式**

适合与 Git 工作流联动。启动后将常驻后台，监听 Webhook 并驱动 Telegram Bot。

```bash
# 以前台方式测试运行
./gantry --mode server --config ./config.yaml

# 生产环境推荐：使用 nohup 挂载后台运行
nohup ./gantry --mode server --config ./config.yaml > server.log 2>&1 &
```

## 🔗 CI/CD 集成示例 (GitHub Actions)
当您的 GitHub Actions 将源码分发（如通过 rsync）到服务器指定目录后，在 Workflow 的最后追加一个步骤，通过 curl 触发本工具：

```yaml
- name: Trigger Deploy Bot Pipeline
  run: |
    curl -X POST http://${{ secrets.SERVER_IP }}:6780/webhook \
         -H "Content-Type: application/json" \
         -d '{
               "branch": "${{ github.ref_name }}",
               "commit_hash": "${{ github.sha }}",
               "commit_message": "${{ github.event.head_commit.message }}",
               "author": "${{ github.actor }}"
             }'
```

## 启动本地私有仓库 测试
- 启动服务
```shell
docker run -d \
  --name registry \
  --restart=always \
  -p 5000:5000 \
  -v /data/docker-registry:/var/lib/registry \
  registry:2
```

- 删除所有悬空镜像 (Dangling images)
```
docker image prune -f
```
