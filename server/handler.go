package server

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"

	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/gin-gonic/gin"
	"github.com/mymmrac/telego"
)

type Handler struct {
	srv *Server
}

// 处理 Git Commit Webhook
func (h *Handler) HandleCommit(c *gin.Context) {
	var payload WebhookPayload

	// ShouldBindJSON 会自动解析 JSON
	// 同时，Gin 会在请求生命周期结束时自动关闭 c.Request.Body
	if err := c.ShouldBindJSON(&payload); err != nil {
		Fail(c, http.StatusBadRequest, 1, "bad request bind failed")
		return
	}

	// 查询分支对应的环境
	envName := h.srv.findEnvByBranch(payload.Repository, payload.Branch)
	if envName == "" {
		Fail(c, http.StatusBadRequest, 2, "Branch not mapped to any environment")
		return
	}

	// 使用 c.Request.Context() 传递链路上下文
	err := h.srv.bot.SendDeploymentMenu(c.Request.Context(), envName, payload.CommitHash, payload.CommitMessage, payload.Author)
	if err != nil {
		log.Printf("TG notification dispatch pipeline failed: %v", err)
		Fail(c, http.StatusInternalServerError, 3, "Internal routing error")
		return
	}

	OK(c, "Pipeline trigger dispatched to Telegram")
}

var alarmTemplate = `🚨 <b>[{{.Env}}]</b>
⏰ <b>告警时间</b> {{.Time}}
📝 <b>告警消息</b>
<pre>{{.Content}}</pre>`

type alarmData struct {
	Alias   string
	Time    string
	Content string
}

func (h *Handler) HandleAlarmNotify(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		Fail(c, http.StatusInternalServerError, 1, "Failed to read request body")
		return
	}

	// 请求头获取环境别名
	envAlias := c.GetHeader("X-Env")
	if envAlias == "" {
		envAlias = "生产环境"
	}

	// 获取北京时间
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc) // 设置当前时间为北京时间
	var alarmData = &alarmData{
		Alias:   envAlias,
		Time:    now.Format("2006-01-02 15:04:05"),
		Content: string(bodyBytes),
	}

	var buf bytes.Buffer
	tpl := template.Must(template.New("alarm").Parse(alarmTemplate))
	if err := tpl.Execute(&buf, alarmData); err != nil {
		log.Printf("Failed to execute template: %v", err)
		Fail(c, http.StatusInternalServerError, 2, "Failed to format message")
		return
	}

	// 加载配置并创建新bot实例
	botName := "alarm"
	cfg := h.srv.cfgMgr.GetTGBotByName(botName)
	if cfg == nil {
		log.Printf("Alarm bot not configured")
		Fail(c, http.StatusInternalServerError, 3, "Alarm bot not configured")
		return
	}

	// 发送到 Telegram
	bot, err := h.srv.getBotByName(botName)
	if err != nil {
		log.Printf("Failed to get alarm bot: %v", err)
		Fail(c, http.StatusInternalServerError, 3, "Failed to get alarm bot")
		return
	}

	msg := tu.Message(tu.ID(cfg.ChatID), buf.String()).WithParseMode(telego.ModeHTML)
	if _, err := bot.SendMessage(c.Request.Context(), msg); err != nil {
		log.Printf("Failed to send Telegram message: %v", err)
		Fail(c, http.StatusInternalServerError, 4, "Failed to send Telegram message")
		return
	}

	OK(c, "Alarm notification dispatched to Telegram")
}
