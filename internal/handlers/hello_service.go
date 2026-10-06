package handlers

import (
	"context"

	hellov1 "github.com/smallStepGiantLeap/hello/client/gen/hello/v1"
)

// HelloService implements hello.v1.HelloService. Generated once by vikrant: this file belongs to the
// service team.
type HelloService struct {
	hellov1.UnimplementedHelloServiceServer
}

// NewHelloService returns the service implementation main registers.
func NewHelloService() *HelloService { return &HelloService{} }

// Hello is unary and marked idempotent in the proto, so the mesh retries it
// on UNAVAILABLE. Keep it safe to run twice.
func (s *HelloService) Hello(ctx context.Context, req *hellov1.HelloRequest) (*hellov1.HelloResponse, error) {
	return &hellov1.HelloResponse{Message: greeting(req.GetName())}, nil
}

// greeting builds the message; split out so the format has one home.
func greeting(name string) string { return "hello " + name }
