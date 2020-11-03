package userimpl

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/google/uuid"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/user"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"strings"
	"time"
)

type resVkOAuth struct {
	VkID        uint32 `json:"user_id"`
	AccessToken string `json:"access_token"`
	Email       string `json:"email"`
}

func makeSafeFastHTTPClient() *fasthttp.Client {
	return &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		ReadTimeout:              5 * time.Second,
		WriteTimeout:             5 * time.Second,
		MaxConnWaitTimeout:       5 * time.Second,
		MaxResponseBodySize:      4 * 1024 * 1024,
		ReadBufferSize:           4 * 1024 * 1024,
	}
}

func makeRedirectURI() string {
	var scheme string
	if config.Env.Host.URL == "leaq.local" {
		scheme = "http://"
	} else {
		scheme = "https://"
	}

	return strings.Join([]string{
		scheme,
		config.Env.Host.URL,
		"/vk-auth",
	}, "")
}

func (*server) VkAuth(ctx context.Context, req *pbUser.VkAuthRequest) (res *pbUser.SelfUser, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if req.GetCode() == "" {
		err = errors.New("code required")
		return
	}

	oAuthURL, err := url.Parse("https://oauth.vk.com/access_token")
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	q := oAuthURL.Query()
	q.Add("client_id", config.Env.Vk.AppID)
	q.Add("client_secret", config.Env.Vk.AppSecretKey)
	q.Add("redirect_uri", makeRedirectURI())
	q.Add("code", req.GetCode())
	oAuthURL.RawQuery = q.Encode()

	reqVk := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(reqVk)
	reqVk.SetRequestURI(oAuthURL.String())

	resVk := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resVk)

	err = makeSafeFastHTTPClient().Do(reqVk, resVk)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	if resVk.StatusCode() != fasthttp.StatusOK {
		err = errors.New("VK response not 200 code")
		logger.Log.Error().Err(err).Send()
		return
	}

	var authUser resVkOAuth
	err = json.Unmarshal(resVk.Body(), &authUser)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	vk := api.NewVK(authUser.AccessToken)
	vk.Limit = api.LimitUserToken

	userByID, err := vk.UsersGet(api.Params{
		"user_ids": authUser.VkID,
		"fields":   "photo_50,photo_200_orig",
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	_, err = mongo.Users.UpdateOne(ctx, user.User{
		VkID: authUser.VkID,
	}, bson.M{
		"$set": user.User{
			VkID:      uint32(userByID[0].ID),
			FirstName: userByID[0].FirstName,
			LastName:  userByID[0].LastName,
			Email:     authUser.Email,
			Photo:     userByID[0].Photo50,
			PhotoRec:  userByID[0].Photo200Orig,
		},
		"$setOnInsert": user.User{
			Token: uuid.New().String(),
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var dbUser user.User
	err = mongo.Users.FindOne(ctx, user.User{
		VkID: authUser.VkID,
	}).Decode(&dbUser)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &pbUser.SelfUser{
		Id:        dbUser.ID.Hex(),
		VkId:      dbUser.VkID,
		Token:     dbUser.Token,
		FirstName: dbUser.FirstName,
		LastName:  dbUser.LastName,
		Photo:     dbUser.Photo,
		PhotoRec:  dbUser.PhotoRec,
	}
	return
}
