package config

import "time"

type AppConfig struct {
	AccessTokenSecret  string
	RefreshTokenSecret string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	AccessTokenIssuer  string
	RefreshTokenIssuer string
	AppEnv             string
	AppPort            string
	
}

func GetConfig() {

}
