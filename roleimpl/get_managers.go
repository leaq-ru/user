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

func (*server) GetManagers(ctx context.Context, req *pbUser.GetManagersRequest) (
	res *pbUser.GetManagersResponse,
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

	auth, err := role.GrantGte(ctx, authUserOID, companyOID, role.Admin)
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

	type userOID = primitive.ObjectID
	grants := map[userOID]role.Grant{}
	var userOIDs []primitive.ObjectID
	for curRoles.Next(ctx) {
		var roleDoc role.Role
		err = curRoles.Decode(&roleDoc)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return
		}

		grants[roleDoc.UserID] = roleDoc.Grant
		userOIDs = append(userOIDs, roleDoc.UserID)
	}

	curUsers, err := mongo.Users.Find(ctx, bson.M{
		"_id": bson.M{
			"$in": userOIDs,
		},
	})
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

	res = &pbUser.GetManagersResponse{}
	for curUsers.Next(ctx) {
		var userDoc user.User
		err = curUsers.Decode(&userDoc)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return
		}

		grant, ok := grants[userDoc.ID]
		if !ok {
			err = errors.New("expected to get grant but nothing found")
			logger.Log.Error().Str("userID", userDoc.ID.Hex()).Err(err).Send()
			return
		}

		res.Managers = append(res.Managers, &pbUser.ManagerItem{
			Id:        userDoc.ID.Hex(),
			FirstName: userDoc.FirstName,
			LastName:  userDoc.LastName,
			Photo:     userDoc.Photo,
			Grant:     pbUser.Grant(grant),
		})
	}
	return
}
