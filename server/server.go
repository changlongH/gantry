package server

import (
	"encoding/json"
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
	Branch        string `json:"branch"`
	CommitHash    string `json:"commit_hash"`
	CommitMessage string `json:"commit_message"`
	Author        string `json:"author"`
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var envName string
	// 查询分支对应的环境
	for name, env := range s.cfgMgr.Get().Envs {
		if env.Branch == payload.Branch {
			envName = name
			break
		}
	}
	if envName == "" {
		http.Error(w, "Branch not mapped to any environment", http.StatusBadRequest)
		return
	}

	/*
		err := s.bot.SendDeploymentMenu(envName, payload.CommitHash, payload.CommitMessage, payload.Author)
		if err != nil {
			log.Printf("TG notification dispatch pipeline failed: %v", err)
			http.Error(w, "Internal routing error", http.StatusInternalServerError)
			return
		}]
	*/

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Pipeline trigger dispatched to Telegram"))
}

func (s *Server) Start() error {
	http.HandleFunc("/webhook", s.handleWebhook)
	return http.ListenAndServe(s.cfgMgr.Get().Server.Address, nil)
}
