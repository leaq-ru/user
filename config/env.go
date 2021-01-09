package config

import (
	"github.com/kelseyhightower/envconfig"
)

const ServiceName = "user"

type c struct {
	Host     host
	Grpc     grpc
	MongoDB  mongodb
	Vk       vk
	Yandex   yandex
	Service  service
	LogLevel string `envconfig:"LOGLEVEL"`
	Dev      dev
}

type host struct {
	URL string `envconfig:"HOST_URL"`
}

type grpc struct {
	Port string `envconfig:"GRPC_PORT"`
}

type mongodb struct {
	URL string `envconfig:"MONGODB_URL"`
}

type vk struct {
	AppID        string `envconfig:"VK_APPID"`
	AppSecretKey string `envconfig:"VK_APPSECRETKEY"`
}

type yandex struct {
	AppID       string `envconfig:"YANDEX_APPID"`
	AppPassword string `envconfig:"YANDEX_APPPASSWORD"`
}

type service struct {
	Parser string `envconfig:"SERVICE_PARSER"`
}

type dev struct {
	BypassCompanyVerify bool `envconfig:"DEV_BYPASSCOMPANYVERIFY"`
}

var Env c

func init() {
	envconfig.MustProcess("", &Env)
}
