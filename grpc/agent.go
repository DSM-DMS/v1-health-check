// Create package in v.1.0.0
// consul package define struct which is implement various interface about gRPC agency using in each of domain
// there are kind of method in gRPC agency such as ping connection check, etc ...

// in agent.go file, define struct type of gRPC agent & initializer that are not method.
// Also if exist, custom type or variable used in common in each of method will declared in this file.

package grpc

// gRPCAgent is struct that agent various command about gRPC including ping for connection check, etc ...
type gRPCAgent struct {}

// NewGRPCAgent return new instance of gRPCAgent pointer type initialized with parameter
func NewGRPCAgent() *gRPCAgent {
	return &gRPCAgent{}
}
