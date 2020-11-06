package role

import (
	"context"
	"errors"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	m "go.mongodb.org/mongo-driver/mongo"
	"time"
)

func GrantGte(ctx context.Context, userID, companyID primitive.ObjectID, minGrant Grant) (ok bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = mongo.Roles.FindOne(ctx, bson.M{
		"u": userID,
		"c": companyID,
		"g": bson.M{
			"$lte": minGrant,
		},
	}).Err()
	if err != nil {
		if errors.Is(err, m.ErrNoDocuments) {
			err = nil
			return
		}
		logger.Log.Error().Err(err).Send()
	} else {
		ok = true
	}
	return
}
