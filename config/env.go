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
	LogLevel string `envconfig:"LOGLEVEL"`
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

var Env c

func init() {
	envconfig.MustProcess("", &Env)
}
