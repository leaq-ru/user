package userimpl

import (
	"context"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func (s *server) GetByIds(ctx context.Context, req *pbUser.GetByIdsRequest) (*pbUser.ShortUsers, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ids := make(bson.A, len(req.GetUserIds()))
	for i, id := range req.GetUserIds() {
		oID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return nil, err
		}
		ids[i] = oID
	}

	cur, err := mongo.Users.Find(ctx, bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return nil, err
	}

	var users []user.User
	err = cur.All(ctx, &users)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return nil, err
	}

	res := &pbUser.ShortUsers{
		Users: make([]*pbUser.ShortUser, len(users)),
	}
	for i, u := range users {
		res.Users[i] = &pbUser.ShortUser{
			Id:        u.ID.Hex(),
			VkId:      u.VkID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Photo:     u.Photo,
			PhotoRec:  u.PhotoRec,
			BanReview: u.BanReview,
		}
	}
	return res, nil
}
