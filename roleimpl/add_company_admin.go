package roleimpl

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	pbUser "github.com/leaq-ru/proto/codegen/go/user"
	"github.com/leaq-ru/user/logger"
	"github.com/leaq-ru/user/md"
	"github.com/leaq-ru/user/mongo"
	"github.com/leaq-ru/user/role"
	"github.com/leaq-ru/user/user"
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

	const maxAdmins = 50
	adminsCount, err := mongo.Roles.CountDocuments(ctx, role.Role{
		CompanyID: companyOID,
		Grant:     role.Admin,
	}, options.Count().SetLimit(maxAdmins))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	if adminsCount >= maxAdmins {
		err = errors.New(fmt.Sprintf("max %v admins", maxAdmins))
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
