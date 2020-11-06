package roleimpl

import (
	"bytes"
	"context"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/call"
	"github.com/nnqq/scr-user/fasthttpclient"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/md"
	"github.com/nnqq/scr-user/mongo"
	"github.com/nnqq/scr-user/orgverify"
	"github.com/nnqq/scr-user/role"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	m "go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
	"net/url"
	"time"
)

func (*server) VerifyCompanyOwner(ctx context.Context, req *user.VerifyCompanyOwnerRequest) (
	res *empty.Empty,
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

	urlToVerify := "http://" + compURL.Host

	var eg errgroup.Group
	var compOID primitive.ObjectID
	eg.Go(func() (e error) {
		comp, e := call.Company.GetBy(ctx, &parser.GetByRequest{
			Url: urlToVerify,
		})
		if e != nil {
			return
		}
		oid, e := primitive.ObjectIDFromHex(comp.Id)
		compOID = oid
		return
	})

	var actualMetaContent string
	eg.Go(func() (e error) {
		actualMetaContent, e = extractMeta(urlToVerify)
		return
	})
	err = eg.Wait()
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var expectedVerify orgverify.OrgVerify
	err = mongo.OrgVerifyPending.FindOne(ctx, orgverify.OrgVerify{
		UserID:    authUserOID,
		CompanyID: compOID,
	}).Decode(&expectedVerify)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	if expectedVerify.MetaContent != actualMetaContent {
		err = errors.New("invalid meta content")
		logger.Log.Error().Str("compID", compOID.Hex()).Err(err).Send()
		return
	}

	sess, err := mongo.Client.StartSession()
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}
	defer sess.EndSession(ctx)

	_, err = sess.WithTransaction(ctx, func(sc m.SessionContext) (_ interface{}, errTx error) {
		var egTx errgroup.Group
		egTx.Go(func() error {
			_, e := mongo.OrgVerifyPending.DeleteMany(sc, orgverify.OrgVerify{
				CompanyID: compOID,
			})
			return e
		})

		egTx.Go(func() error {
			_, e := mongo.OrgVerifySuccess.InsertOne(sc, orgverify.OrgVerify{
				UserID:      authUserOID,
				CompanyID:   compOID,
				MetaContent: actualMetaContent,
				CreatedAt:   time.Now().UTC(),
			})
			return e
		})

		egTx.Go(func() error {
			_, e := mongo.Roles.InsertOne(sc, role.Role{
				UserID:    authUserOID,
				CompanyID: compOID,
				Grant:     role.Owner,
			})
			return e
		})
		errTx = egTx.Wait()
		return
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	res = &empty.Empty{}
	return
}

func extractMeta(urlToVerify string) (actualMetaContent string, err error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(urlToVerify)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	err = fasthttpclient.New().DoRedirects(req, res, 5)
	if err != nil {
		return
	}

	dom, e := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if e != nil {
		return
	}

	dom.Find("meta").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if name, _ := s.Attr("name"); name == orgverify.MetaName {
			content, ok := s.Attr("content")
			if ok {
				actualMetaContent = content
				return false
			}
		}
		return true
	})
	return
}
