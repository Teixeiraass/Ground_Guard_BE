package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config store all configuration of the application.
// The values are read by viper from a config file or environmnet variables.
type Config struct {
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	MQTTEnabled          bool          `mapstructure:"MQTT_ENABLED"`
	MQTTBrokerURL        string        `mapstructure:"MQTT_BROKER_URL"`
	MQTTClientID         string        `mapstructure:"MQTT_CLIENT_ID"`
	MQTTUsername         string        `mapstructure:"MQTT_USERNAME"`
	MQTTPassword         string        `mapstructure:"MQTT_PASSWORD"`
	MQTTTopicPrefix      string        `mapstructure:"MQTT_TOPIC_PREFIX"`
	GoogleClientID       string        `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret   string        `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL    string        `mapstructure:"GOOGLE_REDIRECT_URL"`
	AppleClientID        string        `mapstructure:"APPLE_CLIENT_ID"`
	AppleTeamID          string        `mapstructure:"APPLE_TEAM_ID"`
	AppleKeyID           string        `mapstructure:"APPLE_KEY_ID"`
	ApplePrivateKey      string        `mapstructure:"APPLE_PRIVATE_KEY"`
	AppleRedirectURL     string        `mapstructure:"APPLE_REDIRECT_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.BindEnv("DB_DRIVER")
	viper.BindEnv("DB_SOURCE")
	viper.BindEnv("SERVER_ADDRESS")
	viper.BindEnv("TOKEN_SYMMETRIC_KEY")
	viper.BindEnv("ACCESS_TOKEN_DURATION")
	viper.BindEnv("MQTT_ENABLED")
	viper.BindEnv("MQTT_BROKER_URL")
	viper.BindEnv("MQTT_CLIENT_ID")
	viper.BindEnv("MQTT_USERNAME")
	viper.BindEnv("MQTT_PASSWORD")
	viper.BindEnv("MQTT_TOPIC_PREFIX")
	viper.BindEnv("GOOGLE_CLIENT_ID")
	viper.BindEnv("GOOGLE_CLIENT_SECRET")
	viper.BindEnv("GOOGLE_REDIRECT_URL")
	viper.BindEnv("APPLE_CLIENT_ID")
	viper.BindEnv("APPLE_TEAM_ID")
	viper.BindEnv("APPLE_KEY_ID")
	viper.BindEnv("APPLE_PRIVATE_KEY")
	viper.BindEnv("APPLE_REDIRECT_URL")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	return
}
