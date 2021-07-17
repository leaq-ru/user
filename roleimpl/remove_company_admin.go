package roleimpl

import (
	"context"
	"errors"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/leaq-ru/proto/codegen/go/user"
	"github.com/leaq-ru/user/logger"
	"github.com/leaq-ru/user/md"
	"github.com/leaq-ru/user/mongo"
	"github.com/leaq-ru/user/role"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func (*server) RemoveCompanyAdmin(ctx context.Context, req *user.RemoveCompanyAdminRequest) (
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

	auth, err := role.GrantGte(ctx, authUserOID, companyOID, role.Owner)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	if !auth {
		err = errors.New("unauthorized")
		return
	}

	_, err = mongo.Roles.DeleteOne(ctx, role.Role{
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
