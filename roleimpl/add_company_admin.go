package roleimpl

import (
	"context"
	"errors"
	"github.com/golang/protobuf/ptypes/empty"
	pbUser "github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/md"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/role"
	"github.com/nnqq/scr-user/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
	m "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (*server) AddCompanyAdmin(ctx context.Context, req *pbUser.AddCompanyAdminRequest) (
	res *empty.Empty,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if req.GetCompanyId() == "" {
		err = errors.New("companyId required")
		return
	}
	if req.GetUserId() == "" {
		err = errors.New("userId required")
		return
	}

	userOID, err := primitive.ObjectIDFromHex(req.GetUserId())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	authUserID, err := md.GetUserID(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	if authUserID == req.GetUserId() {
		err = errors.New("can't target yourself")
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

	err = mongo.Users.FindOne(ctx, user.User{
		ID: userOID,
	}).Err()
	if err != nil {
		logger.Log.Error().Err(err).Send()

		if errors.Is(err, m.ErrNoDocuments) {
			err = errors.New("user not found")
		}
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

	adminsCount, err := mongo.Roles.CountDocuments(ctx, role.Role{
		CompanyID: companyOID,
		Grant:     role.Admin,
	}, options.Count().SetLimit(100))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	if adminsCount >= 100 {
		err = errors.New("max 100 admins")
		return
	}

	_, err = mongo.Roles.InsertOne(ctx, role.Role{
		UserID:    userOID,
		CompanyID: companyOID,
		Grant:     role.Admin,
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &empty.Empty{}
	return
}
