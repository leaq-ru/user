package roleimpl

import (
	"bytes"
	"context"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/leaq-ru/proto/codegen/go/parser"
	"github.com/leaq-ru/proto/codegen/go/user"
	"github.com/leaq-ru/user/call"
	"github.com/leaq-ru/user/company_verify"
	"github.com/leaq-ru/user/config"
	"github.com/leaq-ru/user/fasthttpclient"
	"github.com/leaq-ru/user/logger"
	"github.com/leaq-ru/user/md"
	"github.com/leaq-ru/user/mongo"
	"github.com/leaq-ru/user/role"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	m "go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
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

	u := removeURLPrefixes(req.GetCompanyUrl())

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

	var eg errgroup.Group
	var compOID primitive.ObjectID
	eg.Go(func() (e error) {
		comp, e := call.Company.GetBy(ctx, &parser.GetByRequest{
			Url: compURL,
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
		actualMetaContent, e = extractMeta(u)
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
		logger.Log.Error().
			Str("compID", compOID.Hex()).
			Str("expectedVerify.MetaContent", expectedVerify.MetaContent).
			Str("actualMetaContent", actualMetaContent).
			Err(err).
			Send()
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

	_, err = mongo.CompanyVerifyPending.DeleteMany(sc, company_verify.CompanyVerify{
		CompanyID: compOID,
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	_, err = mongo.CompanyVerifySuccess.InsertOne(sc, company_verify.CompanyVerify{
		UserID:      authUserOID,
		CompanyID:   compOID,
		MetaContent: actualMetaContent,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return
	}

	_, err = mongo.Roles.InsertOne(sc, role.Role{
		UserID:    authUserOID,
		CompanyID: compOID,
		Grant:     role.Owner,
	})
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

func extractMeta(rawHost string) (actualMetaContent string, err error) {
	host, err := ensureRfHostIsPunycode(rawHost)
	if err != nil {
		return
	}
	urlToVerify := "http://" + host

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(urlToVerify)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	err = fasthttpclient.New().DoRedirects(req, res, 5)
	if err != nil {
		return
	}

	body := res.Body()
	logger.Log.Debug().Str("body", string(body)).Msg("http body")

	dom, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
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
