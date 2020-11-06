package roleimpl

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/call"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/md"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/orgverify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	m "go.mongodb.org/mongo-driver/mongo"
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

	findErr := mongo.OrgVerifySuccess.FindOne(ctx, orgverify.OrgVerify{
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

	_, err = mongo.OrgVerifyPending.UpdateOne(ctx, orgverify.OrgVerify{
		UserID:    authUserOID,
		CompanyID: compOID,
	}, bson.M{
		"$set": orgverify.OrgVerify{
			MetaContent: uuid.New().String(),
			CreatedAt:   time.Now().UTC(),
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var ov orgverify.OrgVerify
	err = mongo.OrgVerifyPending.FindOne(ctx, orgverify.OrgVerify{
		UserID:    authUserOID,
		CompanyID: compOID,
	}).Decode(&ov)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &user.ApplyCompanyOwnerResponse{
		MetaName:    orgverify.MetaName,
		MetaContent: ov.MetaContent,
	}
	return
}
