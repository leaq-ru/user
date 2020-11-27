package userimpl

import (
	"context"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func (s *server) GetById(ctx context.Context, req *pbUser.GetByIdRequest) (res *pbUser.ShortUser, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	oID, err := primitive.ObjectIDFromHex(req.GetUserId())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var doc user.User
	err = mongo.Users.FindOne(ctx, user.User{
		ID: oID,
	}).Decode(&doc)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &pbUser.ShortUser{
		Id:        doc.ID.Hex(),
		VkId:      doc.VkID,
		FirstName: doc.FirstName,
		LastName:  doc.LastName,
		Photo:     doc.Photo,
		PhotoRec:  doc.PhotoRec,
	}
	return
}
