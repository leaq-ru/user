package userimpl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	pbUser "github.com/leaq-ru/proto/codegen/go/user"
	"github.com/leaq-ru/user/config"
	"github.com/leaq-ru/user/fasthttpclient"
	"github.com/leaq-ru/user/logger"
	"github.com/leaq-ru/user/mongo"
	"github.com/leaq-ru/user/user"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *server) YandexAuth(ctx context.Context, req *pbUser.YandexAuthRequest) (res *pbUser.SelfUser, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if req.GetCode() == "" {
		err = ErrCodeRequired
		return
	}

	ise := errors.New(http.StatusText(http.StatusInternalServerError))

	cl := fasthttpclient.New()

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", req.GetCode())

	reqToken := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(reqToken)
	reqToken.SetRequestURI("https://oauth.yandex.ru/token")
	reqToken.Header.SetMethod(fasthttp.MethodPost)
	reqToken.Header.SetContentType("application/x-www-form-urlencoded")
	reqToken.Header.Set(
		"Authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte(strings.Join([]string{
			config.Env.Yandex.AppID,
			config.Env.Yandex.AppPassword,
		}, ":"))),
	)
	reqToken.SetBodyString(data.Encode())

	resToken := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resToken)

	err = cl.Do(reqToken, resToken)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		err = ise
		return
	}

	var auth resYandexToken
	err = json.Unmarshal(resToken.Body(), &auth)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		err = ise
		return
	}

	reqLogin := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(reqLogin)
	reqLogin.SetRequestURI("https://login.yandex.ru/info")
	reqLogin.Header.Set("Authorization", "OAuth "+auth.AccessToken)

	resLogin := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resLogin)

	err = cl.Do(reqLogin, resLogin)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		err = ise
		return
	}

	var yaUser resYandexLogin
	err = json.Unmarshal(resLogin.Body(), &yaUser)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		err = ise
		return
	}

	_, err = mongo.Users.UpdateOne(ctx, user.User{
		YandexID: yaUser.ID,
	}, bson.M{
		"$set": user.User{
			FirstName: yaUser.FirstName,
			LastName:  yaUser.LastName,
			Email:     yaUser.DefaultEmail,
			Photo:     makeYandexAvatar(yaUser.DefaultAvatarID, "islands-50"),
			PhotoRec:  makeYandexAvatar(yaUser.DefaultAvatarID, "islands-200"),
		},
		"$setOnInsert": user.User{
			Token: uuid.New().String(),
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		err = ise
		return
	}

	var dbUser user.User
	err = mongo.Users.FindOne(ctx, user.User{
		YandexID: yaUser.ID,
	}).Decode(&dbUser)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		err = ise
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

type resYandexToken struct {
	AccessToken string `json:"access_token"`
}

type resYandexLogin struct {
	ID              string `json:"id"`
	DefaultEmail    string `json:"default_email"`
	DefaultAvatarID string `json:"default_avatar_id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
}

func makeYandexAvatar(avatarID string, size string) string {
	return fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/%s", avatarID, size)
}
