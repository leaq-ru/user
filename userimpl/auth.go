package userimpl

import (
	"context"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/user"
	"time"
)

func (*server) Auth(ctx context.Context, req *pbUser.AuthRequest) (
	res *pbUser.SelfUser,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var userDoc user.User
	err = mongo.Users.FindOne(ctx, user.User{
		Token: req.GetToken(),
	}).Decode(&userDoc)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &pbUser.SelfUser{
		Id:        userDoc.ID.Hex(),
		VkId:      userDoc.VkID,
		Token:     userDoc.Token,
		FirstName: userDoc.FirstName,
		LastName:  userDoc.LastName,
		Photo:     userDoc.Photo,
		PhotoRec:  userDoc.PhotoRec,
	}
	return
}
