package roleimpl

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/call"
	"github.com/nnqq/scr-user/company_own_tokens"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"time"
)

func (*server) ApplyCompanyOwner(ctx context.Context, req *user.ApplyCompanyOwnerRequest) (
	res *user.ApplyCompanyOwnerResponse,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if req.GetCompanyUrl() == "" {
		err = errors.New("url required")
		logger.Log.Error().Err(err).Send()
		return
	}

	compURL, err := url.Parse(req.GetCompanyUrl())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	comp, err := call.Company.GetBy(ctx, &parser.GetByRequest{
		Url: "http://" + compURL.Host,
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	compOID, err := primitive.ObjectIDFromHex(comp.Id)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	_, err = mongo.CompanyOwnTokens.UpdateOne(ctx, company_own_tokens.CompanyOwnTokens{
		CompanyID: compOID,
	}, bson.M{
		"$setOnInsert": company_own_tokens.CompanyOwnTokens{
			MetaName:    company_own_tokens.MetaName,
			MetaContent: uuid.New().String(),
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var cot company_own_tokens.CompanyOwnTokens
	err = mongo.CompanyOwnTokens.FindOne(ctx, company_own_tokens.CompanyOwnTokens{
		CompanyID: compOID,
	}).Decode(&cot)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &user.ApplyCompanyOwnerResponse{
		MetaName:    cot.MetaName,
		MetaContent: cot.MetaContent,
	}
	return
}
