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
	"golang.org/x/net/idna"
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

	u := removeURLPrefixes(req.GetCompanyUrl())

	if u == "leaq.ru" {
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

	compURL := "http://" + u

	comp, err := call.Company.GetBy(ctx, &parser.GetByRequest{
		Url: compURL,
	})
	if err != nil {
		if strings.Contains(err.Error(), m.ErrNoDocuments.Error()) {
			var urlToReindex string
			urlToReindex, err = ensureRfHostIsPunycode(u)
			if err != nil {
				logger.Log.Error().Err(err).Send()
				return
			}

			_, err = call.Company.Reindex(ctx, &parser.ReindexRequest{
				Url: urlToReindex,
			})
			if err != nil {
				logger.Log.Error().Err(err).Send()
				return
			}

			comp, err = call.Company.GetBy(ctx, &parser.GetByRequest{
				Url: compURL,
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

func ensureRfHostIsPunycode(rawURL string) (res string, err error) {
	if !strings.HasSuffix(rawURL, ".рф") {
		res = rawURL
		return
	}

	res, err = idna.New().ToASCII(rawURL)
	if err != nil {
		logger.Log.Error().Err(err).Send()
	}
	return
}

func removeURLPrefixes(inURL string) (outURL string) {
	outURL = strings.TrimPrefix(inURL, "https://")
	outURL = strings.TrimPrefix(outURL, "http://")
	outURL = strings.TrimPrefix(outURL, "www.")
	return
}
