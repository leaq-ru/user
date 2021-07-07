package userimpl

import (
	"context"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

func (s *server) ModifyRights(ctx context.Context, req *pbUser.ModifyRightsRequest) (*emptypb.Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	userID, err := primitive.ObjectIDFromHex(req.GetUserId())
	if err != nil {
		return nil, err
	}

	set := bson.M{}
	q := bson.M{
		"$set": set,
	}
	if req.GetBanReview() != nil {
		set["br"] = req.GetBanReview().GetValue()
	}

	_, err = mongo.Users.UpdateOne(ctx, user.User{
		ID: userID,
	}, q)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
