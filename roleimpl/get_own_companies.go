package roleimpl

import (
	"context"
	"github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/role"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (*server) GetOwnCompanies(ctx context.Context, req *user.GetOwnCompaniesRequest) (
	res *user.GetOwnCompaniesResponse,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cur, err := mongo.Roles.Find(ctx, bson.M{
		"u": req.GetUserId(),
	}, options.Find().
		SetSort(bson.M{
			"c": -1,
		}).
		SetSkip(int64(req.GetOpts().GetSkip())).
		SetLimit(int64(req.GetOpts().GetLimit())))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &user.GetOwnCompaniesResponse{}
	for cur.Next(ctx) {
		var roleItem role.Role
		err = cur.Decode(&roleItem)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return
		}
		res.CompanyIds = append(res.CompanyIds, roleItem.CompanyID.Hex())
	}
	return
}
