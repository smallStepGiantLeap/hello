package handlers

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	hellov1 "github.com/smallStepGiantLeap/hello/client/gen/hello/v1"
)

func TestHello(t *testing.T) {
	for name, want := range map[string]string{"Subramanian": "hello Subramanian", "  Ada ": "hello Ada"} {
		got, err := NewHelloService().Hello(context.Background(), &hellov1.HelloRequest{Name: name})
		if err != nil || got.GetMessage() != want {
			t.Errorf("Hello(%q) = %q, %v; want %q", name, got.GetMessage(), err, want)
		}
	}
}

func TestHelloRejectsEmptyName(t *testing.T) {
	_, err := NewHelloService().Hello(context.Background(), &hellov1.HelloRequest{Name: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Hello(blank) = %v; want InvalidArgument", err)
	}
}
