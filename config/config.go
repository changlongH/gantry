package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
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
	Desc             string            `mapstructure:"desc"`               // 环境描述信息
	ProjectPath      string            `mapstructure:"project_path"`       // 项目源码路径
	BuildPath        string            `mapstructure:"build_path"`         // 编译产物路径
	ComposeFile      string            `mapstructure:"compose_file"`       // Docker Compose 文件路径
	Registry         string            `mapstructure:"registry"`           // 镜像仓库地址，留空表示不推送
	Branch           string            `mapstructure:"branch"`             // 当前环境对应的 Git 分支
	Dockerfile       string            `mapstructure:"dockerfile"`         // 可选项，默认为项目根目录下的 Dockerfile 相对路径
	BuildArgs        map[string]string `mapstructure:"build_args"`         // 构建扩展参数
	ImageTagStrategy string            `mapstructure:"image_tag_strategy"` // 镜像标签策略
}

// Manager 负责并发安全地管理配置
type Manager struct {
	mu  sync.RWMutex
	cfg *Config

	serviceCache map[string][]string // 局部缓存：key为 envName，value为该环境下的服务列表
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
		mgr.serviceCache = make(map[string][]string) // 配置变更后清空服务列表缓存
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

// GetAppServices 获取指定环境下的服务列表
func (m *Manager) GetAppServices(envName string) []string {
	cfg := m.Get()
	return cfg.Apps
}

// GetDockerComposeServices 动态解析指定环境的 Compose 文件获取真实服务列表。
func (m *Manager) GetDockerComposeServices(envName string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 如果缓存中已有数据，直接返回缓存结果
	if list, exists := m.serviceCache[envName]; exists {
		return list
	}

	// 2. 查找环境配置
	env, ok := m.cfg.Envs[envName]
	if !ok || env.ComposeFile == "" {
		return nil
	}

	// 3. 动态解析对应的 docker-compose.yaml
	services, err := parseComposeServices(env.ComposeFile)
	if err != nil {
		log.Printf("⚠️ Failed to parse compose file [%s] for env [%s]: %v.", env.ComposeFile, envName, err)
		return nil
	}

	// 4. 将成功解析的结果写入缓存
	m.serviceCache[envName] = services
	return services
}

// 内部私有辅助函数：专门负责反序列化 yaml
func parseComposeServices(filePath string) ([]string, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var composeData struct {
		Services map[string]interface{} `yaml:"services"`
	}

	if err := yaml.Unmarshal(fileBytes, &composeData); err != nil {
		return nil, err
	}

	services := make([]string, 0, len(composeData.Services))
	for svcName := range composeData.Services {
		services = append(services, svcName)
	}

	return services, nil
}
