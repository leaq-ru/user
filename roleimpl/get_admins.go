package roleimpl

import (
	"context"
	"errors"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/md"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/role"
	"github.com/nnqq/scr-user/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (*server) GetAdmins(ctx context.Context, req *pbUser.GetAdminsRequest) (
	res *pbUser.GetAdminsResponse,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if req.GetCompanyId() == "" {
		err = errors.New("companyId required")
		return
	}

	limit := int64(20)
	if req.GetOpts() != nil {
		if req.GetOpts().GetLimit() > 100 || req.GetOpts().GetLimit() < 0 {
			err = errors.New("limit out of 1-100")
			return
		} else if req.GetOpts().GetLimit() != 0 {
			limit = int64(req.GetOpts().GetLimit())
		}
	}

	authUserID, err := md.GetUserID(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	authUserOID, err := primitive.ObjectIDFromHex(authUserID)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	companyOID, err := primitive.ObjectIDFromHex(req.GetCompanyId())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	auth, err := role.GrantGte(ctx, authUserOID, companyOID, role.Owner)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	if !auth {
		err = errors.New("unauthorized")
		return
	}

	curRoles, err := mongo.Roles.Find(ctx, role.Role{
		CompanyID: companyOID,
		Grant:     role.Admin,
	}, options.Find().
		SetSort(bson.M{
			"u": -1,
		}).
		SetSkip(int64(req.GetOpts().GetSkip())).
		SetLimit(limit))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	defer func() {
		e := curRoles.Close(ctx)
		if e != nil {
			logger.Log.Error().Err(e).Send()
		}
	}()

	var roleOIDs []primitive.ObjectID
	for curRoles.Next(ctx) {
		var roleDoc role.Role
		err = curRoles.Decode(&roleDoc)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return
		}

		roleOIDs = append(roleOIDs, roleDoc.UserID)
	}

	curUsers, err := mongo.Users.Find(ctx, bson.M{
		"_id": bson.M{
			"$in": roleOIDs,
		},
	}, options.Find().SetSort(bson.M{
		"_id": -1,
	}))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	defer func() {
		e := curUsers.Close(ctx)
		if e != nil {
			logger.Log.Error().Err(e).Send()
		}
	}()

	res = &pbUser.GetAdminsResponse{}
	for curUsers.Next(ctx) {
		var userDoc user.User
		err = curUsers.Decode(&userDoc)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return
		}

		res.Admins = append(res.Admins, &pbUser.ShortUser{
			Id:        userDoc.ID.Hex(),
			FirstName: userDoc.FirstName,
			LastName:  userDoc.LastName,
			Photo:     userDoc.Photo,
		})
	}
	return
}
