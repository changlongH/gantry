package config

import (
	"fmt"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Telegram TelegramConfig         `mapstructure:"telegram"`
	Server   ServerConfig           `mapstructure:"server"`
	Envs     map[string]Environment `mapstructure:"environments"`
	Apps     []string               `mapstructure:"apps"`
}

type TelegramConfig struct {
	Token  string `mapstructure:"token"`
	ChatID int64  `mapstructure:"chat_id"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"`
	Secret  string `mapstructure:"secret"`
}

type Environment struct {
	Desc             string            `mapstructure:"desc"` // 环境描述信息
	ProjectPath      string            `mapstructure:"project_path"`
	ComposeFile      string            `mapstructure:"compose_file"`
	Registry         string            `mapstructure:"registry"`
	Branch           string            `mapstructure:"branch"`
	Dockerfile       string            `mapstructure:"dockerfile"`
	BuildArgs        map[string]string `mapstructure:"build_args"`
	ImageTagStrategy string            `mapstructure:"image_tag_strategy"`
}

// Manager 负责并发安全地管理配置
type Manager struct {
	mu  sync.RWMutex
	cfg *Config
}

func InitManager(path string) (*Manager, error) {
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	mgr := &Manager{cfg: &cfg}

	// 开启 Viper 的配置监听机制
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("🔄 Config file modified: %s. Reloading...", e.Name)

		var newCfg Config
		if err := viper.Unmarshal(&newCfg); err != nil {
			log.Printf("❌ Hot reload failed (invalid format): %v", err)
			return
		}

		// 通过写锁安全替换全局配置指针
		mgr.mu.Lock()
		mgr.cfg = &newCfg
		mgr.mu.Unlock()

		log.Println("✅ Config hot-reloaded successfully.")
	})

	return mgr, nil
}

// Get 获取当前最新配置的只读快照
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}
