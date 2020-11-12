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
	"github.com/nnqq/scr-user/company_verify"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/fasthttpclient"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/md"
	"github.com/nnqq/scr-user/mongo"
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

	compURL, err := url.Parse("http://" + req.GetCompanyUrl())
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var eg errgroup.Group
	var compOID primitive.ObjectID
	eg.Go(func() (e error) {
		comp, e := call.Company.GetBy(ctx, &parser.GetByRequest{
			Url: compURL.String(),
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
		actualMetaContent, e = extractMeta(compURL.String())
		return
	})
	err = eg.Wait()
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	var expectedVerify company_verify.CompanyVerify
	err = mongo.CompanyVerifyPending.FindOne(ctx, company_verify.CompanyVerify{
		UserID:    authUserOID,
		CompanyID: compOID,
	}).Decode(&expectedVerify)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	if !config.Env.Dev.BypassCompanyVerify && expectedVerify.MetaContent != actualMetaContent {
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

	err = sess.StartTransaction()
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	sc := m.NewSessionContext(ctx, sess)
	var egTx errgroup.Group
	egTx.Go(func() error {
		_, e := mongo.CompanyVerifyPending.DeleteMany(sc, company_verify.CompanyVerify{
			CompanyID: compOID,
		})
		return e
	})

	egTx.Go(func() error {
		_, e := mongo.CompanyVerifySuccess.InsertOne(sc, company_verify.CompanyVerify{
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
	err = egTx.Wait()
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	err = sess.CommitTransaction(sc)
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
		if name, _ := s.Attr("name"); name == company_verify.MetaName {
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
