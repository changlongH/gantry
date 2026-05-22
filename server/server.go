package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/changlongH/gantry/config"
	"github.com/changlongH/gantry/tgbot"
)

type Server struct {
	cfgMgr *config.Manager
	bot    *tgbot.Bot
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

// 辅助方法
func (s *Server) findEnvByBranch(repo, branch string) string {
	for name, envCfg := range s.cfgMgr.Get().Envs {
		if envCfg.Git.Branch == branch && envCfg.Git.Repository == repo {
			return name
		}
	}
	return ""
}

func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var secret = s.cfgMgr.Get().Server.Secret
	if secret != "" {
		reqSecret := r.Header.Get("X-Secret")
		if reqSecret != secret {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// 查询分支对应的环境
	var envName = s.findEnvByBranch(payload.Repository, payload.Branch)
	if envName == "" {
		http.Error(w, "Branch not mapped to any environment", http.StatusBadRequest)
		return
	}

	err := s.bot.SendDeploymentMenu(r.Context(), envName, payload.CommitHash, payload.CommitMessage, payload.Author)
	if err != nil {
		log.Printf("TG notification dispatch pipeline failed: %v", err)
		http.Error(w, "Internal routing error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Pipeline trigger dispatched to Telegram"))
}

func (s *Server) Start() error {
	http.HandleFunc("/git/commit", s.handleCommit)
	return http.ListenAndServe(s.cfgMgr.Get().Server.Address, nil)
}
