package roleimpl

import "github.com/leaq-ru/proto/codegen/go/user"

type server struct {
	user.UnimplementedRoleServer
}

func NewServer() *server {
	return &server{}
}
