package server

import (
	"fmt"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/mymmrac/telego"

	"github.com/changlongH/gantry/pkg/config"
	"github.com/changlongH/gantry/tgbot"
)

type Server struct {
	cfgMgr *config.Manager
	bot    *tgbot.Bot // 监听的bot

	botMu sync.Mutex
	bots  map[string]*telego.Bot // 其他bot实例
}

func NewServer(cfgMgr *config.Manager, bot *tgbot.Bot) *Server {
	return &Server{cfgMgr: cfgMgr, bot: bot}
}

type WebhookPayload struct {
	Repository    string `json:"repository"` // 仓库名称例如 "myorg/myrepo"
	Branch        string `json:"branch"`
	CommitHash    string `json:"commit_hash"`
	CommitMessage string `json:"commit_message"`
	Author        string `json:"author"`
}

// 获取监听的listen bot实例
func (s *Server) getListenBot() *tgbot.Bot {
	return s.bot
}

// 获取指定名称的bot实例
func (s *Server) getBotByName(name string) (*telego.Bot, error) {
	s.botMu.Lock()
	bot, exists := s.bots[name]
	s.botMu.Unlock()
	if exists {
		return bot, nil
	}

	// 加载配置并创建新bot实例
	cfg := s.cfgMgr.GetTGBotByName(name)
	if cfg == nil {
		return nil, fmt.Errorf("not found bot")
	}

	bot, err := telego.NewBot(cfg.Token)
	if err != nil {
		return nil, err
	}
	s.botMu.Lock()
	s.bots[name] = bot
	s.botMu.Unlock()
	return bot, nil
}

// 辅助方法
func (s *Server) findEnvByBranch(repo, branch string) string {
	for name, envCfg := range s.cfgMgr.Get().Envs {
		if envCfg.Git.Branch == branch && envCfg.Git.Repository == repo {
			return name
		}
	}
	return ""
}

func (s *Server) Start() error {
	// 设置为生产模式以隐藏调试日志（可选，调试时可注释掉）
	gin.SetMode(gin.ReleaseMode)
	// 初始化 Gin 引擎
	// gin.Default() 默认的日志和崩溃恢复中间件
	r := gin.Default()

	handler := &Handler{
		srv: s,
	}

	// 统一注册带有【鉴权+限流中间件】的 POST 路由
	api := r.Group("/")
	api.Use(s.authAndLimitMiddleware())
	{
		api.POST("/git/commit", handler.HandleCommit)
		api.POST("/alarm/notify", handler.HandleAlarmNotify)
	}

	// 启动监听
	address := s.cfgMgr.Get().Server.Address
	log.Printf("Server is running on %s", address)
	return r.Run(address)
}
