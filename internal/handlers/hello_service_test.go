package handlers

import (
	"context"
	"testing"

	hellov1 "github.com/smallStepGiantLeap/hello/client/gen/hello/v1"
)

func TestHello(t *testing.T) {
	for name, want := range map[string]string{"Subramanian": "hello Subramanian", "": "hello "} {
		got, err := NewHelloService().Hello(context.Background(), &hellov1.HelloRequest{Name: name})
		if err != nil || got.GetMessage() != want {
			t.Errorf("Hello(%q) = %q, %v; want %q", name, got.GetMessage(), err, want)
		}
	}
}
