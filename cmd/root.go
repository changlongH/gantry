package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/changlongH/gantry/pkg/config"
	"github.com/changlongH/gantry/server"
	"github.com/changlongH/gantry/tgbot"
)

func Execute() {
	var mode string
	var cfgPath string

	flag.StringVar(&mode, "mode", "cli", "Running mode: 'cli' or 'server'")
	flag.StringVar(&cfgPath, "config", "config.yaml", "Path to config file")
	flag.Parse()

	// 使用 Viper 初始化并发安全的配置管理器
	cfgMgr, err := config.InitManager(cfgPath)
	if err != nil {
		log.Fatalf("Failed to initialize system config: %v", err)
	}

	switch mode {
	case "cli":
		cliWorkflow := NewInteractiveCLI(cfgMgr)
		cliWorkflow.Run()

	case "server":
		fmt.Println("Initializing automated deployment daemon with Viper Hot-Reload...")
		bot, err := tgbot.NewBot(cfgMgr)
		if err != nil {
			log.Fatalf("Failed to establish Telegram Link: %v", err)
		}
		go bot.Start()

		srv := server.NewServer(cfgMgr, bot)
		log.Printf("Listening for VCS Webhooks on %s", cfgMgr.Get().Server.Address)
		if err := srv.Start(); err != nil {
			log.Fatalf("Daemon failure: %v", err)
		}

	default:
		fmt.Printf("Unknown runtime mode: '%s'. Supported: 'cli' | 'server'\n", mode)
		os.Exit(1)
	}
}
