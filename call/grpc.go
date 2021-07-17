package call

import (
	"github.com/leaq-ru/proto/codegen/go/parser"
	"github.com/leaq-ru/user/config"
	"github.com/leaq-ru/user/logger"
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
