package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config структура для конфигурации приложения
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Redis  RedisConfig  `mapstructure:"redis"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LoadConfig загружает конфигурацию из файла и переменных окружения
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/event-aggregator")

	// Устанавливаем значения по умолчанию
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("redis.addr", "")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Читаем переменные окружения
	viper.AutomaticEnv()
	viper.SetEnvPrefix("APP")
	
	// Для вложенных структур
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Пытаемся прочитать конфигурационный файл
	if err := viper.ReadInConfig(); err != nil {
		// Если файл не найден, используем значения по умолчанию и переменные окружения
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Warning: Could not read config file: %v", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}