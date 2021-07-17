package userimpl

import "github.com/leaq-ru/proto/codegen/go/user"

type server struct {
	user.UnimplementedUserServer
}

func NewServer() *server {
	return &server{}
}
