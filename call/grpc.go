package call

import (
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/logger"
	"google.golang.org/grpc"
)

var (
	Company parser.CompanyClient
)

func init() {
	connParser, err := grpc.Dial(config.Env.Service.Parser, grpc.WithInsecure())
	logger.Must(err)
	Company = parser.NewCompanyClient(connParser)
}
