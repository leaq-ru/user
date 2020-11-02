package vk

import (
	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/nnqq/scr-user/config"
	"strings"
)

var UserApi *api.VK

func init() {
	UserApi = api.NewVK(strings.Split(config.Env.Vk.UserTokens, ",")...)
	UserApi.Limit = api.LimitUserToken
}
