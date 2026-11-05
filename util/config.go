package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config store all configuration of the application.
// The values are read by viper from a config file or environmnet variables.
type Config struct {
	DBDriver                string        `mapstructure:"DB_DRIVER"`
	DBSource                string        `mapstructure:"DB_SOURCE"`
	ServerAddress           string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey       string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration     time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration    time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	MQTTEnabled             bool          `mapstructure:"MQTT_ENABLED"`
	MQTTBrokerURL           string        `mapstructure:"MQTT_BROKER_URL"`
	MQTTClientID            string        `mapstructure:"MQTT_CLIENT_ID"`
	MQTTUsername            string        `mapstructure:"MQTT_USERNAME"`
	MQTTPassword            string        `mapstructure:"MQTT_PASSWORD"`
	MQTTTopicPrefix         string        `mapstructure:"MQTT_TOPIC_PREFIX"`
	GoogleClientID          string        `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret      string        `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL       string        `mapstructure:"GOOGLE_REDIRECT_URL"`
	AppleClientID           string        `mapstructure:"APPLE_CLIENT_ID"`
	AppleTeamID             string        `mapstructure:"APPLE_TEAM_ID"`
	AppleKeyID              string        `mapstructure:"APPLE_KEY_ID"`
	ApplePrivateKey         string        `mapstructure:"APPLE_PRIVATE_KEY"`
	AppleRedirectURL        string        `mapstructure:"APPLE_REDIRECT_URL"`
	FirebaseCredentialsFile string        `mapstructure:"FIREBASE_CREDENTIALS_FILE"`
	MinimumAppVersion       string        `mapstructure:"MINIMUM_APP_VERSION"`
	LatestAppVersion        string        `mapstructure:"LATEST_APP_VERSION"`
	AndroidStoreUrl         string        `mapstructure:"ANDROID_STORE_URL"`
	IosStoreUrl             string        `mapstructure:"IOS_STORE_URL"`
	AppBaseURL              string        `mapstructure:"APP_BASE_URL"`
	SMTPHost                string        `mapstructure:"SMTP_HOST"`
	SMTPPort                string        `mapstructure:"SMTP_PORT"`
	SMTPUsername            string        `mapstructure:"SMTP_USERNAME"`
	SMTPPassword            string        `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom                string        `mapstructure:"SMTP_FROM"`
	TurnstileSecretKey      string        `mapstructure:"TURNSTILE_SECRET_KEY"`
	TurnstileSiteKey        string        `mapstructure:"TURNSTILE_SITE_KEY"`
	FirmwareStorageDir      string        `mapstructure:"FIRMWARE_STORAGE_DIR"`
	FirmwareSigningKeyFile  string        `mapstructure:"FIRMWARE_SIGNING_KEY_FILE"`
	OpenWeatherAPIKey       string        `mapstructure:"OPENWEATHER_API_KEY"`
	// Comma-separated proxy IPs/CIDRs allowed to set X-Forwarded-For.
	TrustedProxies   string `mapstructure:"TRUSTED_PROXIES"`
	RateLimitEnabled bool   `mapstructure:"RATE_LIMIT_ENABLED"`
	// Browser origins allowed to open the websocket (apps send no Origin).
	AllowedOrigins string `mapstructure:"ALLOWED_ORIGINS"`
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
	viper.BindEnv("FIREBASE_CREDENTIALS_FILE")
	viper.BindEnv("MINIMUM_APP_VERSION")
	viper.BindEnv("LATEST_APP_VERSION")
	viper.BindEnv("ANDROID_STORE_URL")
	viper.BindEnv("IOS_STORE_URL")
	viper.BindEnv("APP_BASE_URL")
	viper.BindEnv("SMTP_HOST")
	viper.BindEnv("SMTP_PORT")
	viper.BindEnv("SMTP_USERNAME")
	viper.BindEnv("SMTP_PASSWORD")
	viper.BindEnv("SMTP_FROM")
	viper.BindEnv("TURNSTILE_SECRET_KEY")
	viper.BindEnv("TURNSTILE_SITE_KEY")
	viper.BindEnv("FIRMWARE_STORAGE_DIR")
	viper.SetDefault("FIRMWARE_STORAGE_DIR", "./firmware")
	viper.BindEnv("FIRMWARE_SIGNING_KEY_FILE")
	viper.BindEnv("OPENWEATHER_API_KEY")
	viper.BindEnv("TRUSTED_PROXIES")
	viper.BindEnv("RATE_LIMIT_ENABLED")
	viper.SetDefault("RATE_LIMIT_ENABLED", true)
	viper.BindEnv("ALLOWED_ORIGINS")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	return
}
