package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Database DatabaseConfig `mapstructure:"database"`
	App      AppConfig      `mapstructure:"app"`
}

type TelegramConfig struct {
	BotToken    string  `mapstructure:"bot_token"`
	GroupChatID int64   `mapstructure:"group_chat_id"`
	UserIDs     []int64 `mapstructure:"user_ids"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type AppConfig struct {
	RunOnce      bool   `mapstructure:"run_once"`
	LogLevel     string `mapstructure:"log_level"`
	DaysAhead    int    `mapstructure:"days_ahead"`
	ScheduleCron string `mapstructure:"schedule_cron"`
}

var cfg *Config

func LoadConfig() (*Config, error) {
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("app.run_once", false)
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.days_ahead", 5)
	viper.SetDefault("app.schedule_cron", "0 8 * * *")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
		}
		log.Println("Конфигурационный файл не найден, используем переменные окружения")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("TG")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.BindEnv("telegram.bot_token")
	viper.BindEnv("telegram.group_chat_id")
	viper.BindEnv("telegram.user_ids")
	viper.BindEnv("database.host")
	viper.BindEnv("database.port")
	viper.BindEnv("database.user")
	viper.BindEnv("database.password")
	viper.BindEnv("database.dbname")
	viper.BindEnv("app.run_once")
	viper.BindEnv("app.days_ahead")
	viper.BindEnv("app.schedule_cron")

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("ошибка распаковки конфигурации: %w", err)
	}

	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		if err := SaveConfig(cfg); err != nil {
			log.Printf("Предупреждение: не удалось сохранить конфигурацию: %v", err)
		}
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	viper.Set("telegram", cfg.Telegram)
	viper.Set("database", cfg.Database)
	viper.Set("app", cfg.App)

	if err := viper.SafeWriteConfigAs("config.yaml"); err != nil {
		if os.IsExist(err) {
			return viper.WriteConfig()
		}
		return err
	}
	return nil
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func GetConfig() *Config {
	return cfg
}
