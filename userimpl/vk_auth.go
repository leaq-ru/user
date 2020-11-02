package userimpl

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/google/uuid"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/user"
	"github.com/nnqq/scr-user/vk"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
	"time"
)

func validateVkAuthHash(vkId uint32, inHash string) (valid bool) {
	sum := md5.Sum([]byte(config.Env.Vk.AppID + strconv.Itoa(int(vkId)) + config.Env.Vk.AppSecretKey))
	expectedHash := hex.EncodeToString(sum[:])
	return expectedHash == inHash
}

func (*server) VkAuth(ctx context.Context, req *pbUser.VkAuthRequest) (res *pbUser.SelfUser, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if req.GetFirstName() == "" || req.GetHash() == "" || req.GetLastName() == "" || req.GetPhoto() == "" ||
		req.GetPhotoRec() == "" || req.GetUid() == 0 {
		err = errors.New("invalid payload")
		return
	}

	if !validateVkAuthHash(req.GetUid(), req.GetHash()) {
		err = errors.New("invalid hash")
		logger.Log.Error().Uint32("vkID", req.GetUid()).Str("hash", req.GetHash()).Err(err).Send()
		return
	}

	vkUser, err := vk.UserApi.UsersGet(api.Params{
		"user_ids": req.GetUid(),
		"fields":   "photo_50,photo_200_orig",
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	_, err = mongo.Users.UpdateOne(ctx, user.User{
		VkID: req.GetUid(),
	}, bson.M{
		"$set": user.User{
			FirstName: vkUser[0].FirstName,
			LastName:  vkUser[0].LastName,
			Photo:     vkUser[0].Photo50,
			PhotoRec:  vkUser[0].Photo200Orig,
		},
		"$setOnInsert": user.User{
			Token: uuid.New().String(),
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var u user.User
	err = mongo.Users.FindOne(ctx, user.User{
		VkID: req.GetUid(),
	}).Decode(&u)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &pbUser.SelfUser{
		Id:        u.ID.Hex(),
		VkId:      u.VkID,
		Token:     u.Token,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Photo:     u.Photo,
		PhotoRec:  u.PhotoRec,
	}
	return
}
