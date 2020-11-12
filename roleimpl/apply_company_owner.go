package roleimpl

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/call"
	"github.com/nnqq/scr-user/company_verify"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/md"
	"github.com/nnqq/scr-user/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	m "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"strings"
	"time"
)

func (*server) ApplyCompanyOwner(ctx context.Context, req *user.ApplyCompanyOwnerRequest) (
	res *user.ApplyCompanyOwnerResponse,
	err error,
) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if req.GetCompanyUrl() == "" {
		err = errors.New("url required")
		logger.Log.Error().Err(err).Send()
		return
	}

	if req.GetCompanyUrl() == "leaq.ru" {
		err = errors.New("url invalid")
		logger.Log.Error().Err(err).Send()
		return
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

	compURL, err := url.Parse("http://" + req.GetCompanyUrl())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	comp, err := call.Company.GetBy(ctx, &parser.GetByRequest{
		Url: compURL.String(),
	})
	if err != nil {
		if strings.Contains(err.Error(), m.ErrNoDocuments.Error()) {
			_, errReindex := call.Company.Reindex(ctx, &parser.ReindexRequest{
				Url: compURL.Host,
			})
			if errReindex != nil {
				err = errReindex
				logger.Log.Error().Err(err).Send()
				return
			}

			comp, err = call.Company.GetBy(ctx, &parser.GetByRequest{
				Url: compURL.String(),
			})
			if err != nil {
				logger.Log.Error().Err(err).Send()
				err = errors.New("company not found")
				return
			}
		} else {
			logger.Log.Error().Err(err).Send()
			return
		}
	}
	compOID, err := primitive.ObjectIDFromHex(comp.Id)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	findErr := mongo.CompanyVerifySuccess.FindOne(ctx, company_verify.CompanyVerify{
		CompanyID: compOID,
	}).Err()
	if findErr == nil {
		err = errors.New("company already has owner")
		logger.Log.Error().Str("CompanyID", comp.Id).Err(err).Send()
		return
	}
	if !errors.Is(findErr, m.ErrNoDocuments) {
		err = findErr
		logger.Log.Error().Err(err).Send()
		return
	}

	_, err = mongo.CompanyVerifyPending.UpdateOne(ctx, company_verify.CompanyVerify{
		UserID:    authUserOID,
		CompanyID: compOID,
	}, bson.M{
		"$set": company_verify.CompanyVerify{
			MetaContent: uuid.New().String(),
			CreatedAt:   time.Now().UTC(),
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var cv company_verify.CompanyVerify
	err = mongo.CompanyVerifyPending.FindOne(ctx, company_verify.CompanyVerify{
		UserID:    authUserOID,
		CompanyID: compOID,
	}).Decode(&cv)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &user.ApplyCompanyOwnerResponse{
		MetaName:    company_verify.MetaName,
		MetaContent: cv.MetaContent,
	}
	return
}
