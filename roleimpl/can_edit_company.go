package roleimpl

import (
	"context"
	"errors"
	pbUser "github.com/leaq-ru/proto/codegen/go/user"
	"github.com/leaq-ru/user/logger"
	"github.com/leaq-ru/user/role"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func (*server) CanEditCompany(ctx context.Context, req *pbUser.CanEditCompanyRequest) (
	res *pbUser.CanEditCompanyResponse,
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

	companyOID, err := primitive.ObjectIDFromHex(req.GetCompanyId())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &pbUser.CanEditCompanyResponse{}
	res.CanEdit, err = role.GrantGte(ctx, userOID, companyOID, role.Admin)
	if err != nil {
		logger.Log.Error().Err(err).Send()
	}
	return
}
