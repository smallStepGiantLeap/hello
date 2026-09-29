package handlers

import (
	"context"

	hellov1 "github.com/smallStepGiantLeap/hello/client/gen/hello/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	return nil, status.Error(codes.Unimplemented, "hello.v1.HelloService/Hello is not implemented yet")
}
