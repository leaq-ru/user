package userimpl

import "github.com/nnqq/scr-proto/codegen/go/user"

type server struct {
	user.UnimplementedUserServer
}

func NewServer() *server {
	return &server{}
}
