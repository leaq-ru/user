package roleimpl

import "github.com/nnqq/scr-proto/codegen/go/user"

type server struct {
	user.UnimplementedRoleServer
}

func NewServer() *server {
	return &server{}
}
