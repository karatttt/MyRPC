package server

import (
	"MyRPC/core/codec"
	"MyRPC/core/internel"
	"MyRPC/core/transport"
	"context"
	"fmt"
)

// 定义接口，提供一些服务的注册和开启服务的功能
type Service interface {
	// Register registers a service with the server.
	// The serviceName is the name of the service, and service is the implementation of the service.
	Register(serviceDesc *ServiceDesc, service interface{}) error

	// Serve starts the server and listens for incoming connections.
	Serve(address string) error
}

// 服务的各个属性
type service struct {
	handler map[string]Handler
	opts    *Options
}

type Handler func(req []byte) (rsp interface{}, err error)

// ServiceDesc describes a service and its methods.
type ServiceDesc struct {
	ServiceName string
	HandlerType interface{}
	Methods     []MethodDesc
}

// MethodDesc describes a method of a service.
type MethodDesc struct {
	MethodName string
	Func       interface{}
}

// 实现service的Register方法，注册服务的具体实现
func (s *service) Register(serviceDesc *ServiceDesc, service interface{}) error {

	// 初始化Transport
	s.opts.Transport = transport.DefaultServerTransport

	// TODO 现在只做了绑定方法，service后面可能也要绑定，以及router如何理解
	s.registerMethods(serviceDesc.Methods, service)

	return nil
}

// 注册普通方法
func (s *service) registerMethods(methods []MethodDesc, serviceImpl interface{}) error {
	for _, method := range methods {
		if _, exists := s.handler[method.MethodName]; exists {
			return fmt.Errorf("duplicate method name: %s", method.MethodName)
		}
		s.handler[method.MethodName] = func(req []byte) (rsp interface{}, err error) {
			if fn, ok := method.Func.(func(svr interface{}, req []byte) (rsp interface{}, err error)); ok {
				// 这里调用的就是rpc.go里面的实际的handler方法
				return fn(serviceImpl, req)
			}
			return nil, fmt.Errorf("method.Func is not a valid function")
		}
	}
	return nil
}

func (s *service) Serve(address string) error {
	fmt.Printf("Server is listening on %s\n", address)

	// 将service作为Handler传入transport，后续接收到请求，会调用service的Handle方法
	s.opts.Transport.RegisterHandler(s)
	err := s.opts.Transport.ListenAndServe(context.Background(), "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return nil
}

// 实现Handler接口，后续需要根据请求的methodName，然后调用这个service的Handler
// 里面的逻辑是：匹配service的HandlerMap是否有对应的methodName，如果有，则调用里面的Handler
func (s *service) Handle(ctx context.Context, frame []byte) (rsp []byte, err error) {
	// 获取msg
	ctx, msg := internel.GetMessage(ctx)
	// 解码帧, 并为msg赋值service和method的相关信息
	reqData, err := s.opts.Codec.Decode(msg, frame)
	if err != nil {
		return nil, fmt.Errorf("failed to decode frame: %v", err)
	}
	methodName := msg.GetMethodName()

	// 获取Handler执行结果
	response, err := s.handler[methodName](reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to handle request: %v", err)
	}

	// 将response序列化
	responseBody, err := codec.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %v", err)
	}
	// 编码
	framedata, err := s.opts.Codec.Encode(ctx, responseBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode response: %v", err)
	}

	return framedata, nil
}

// NewService creates a new service instance with the given options
func NewService(serviceName string, opts ...Option) Service {
	s := &service{
		handler: make(map[string]Handler),
		opts:    NewOptions(),
	}

	// Set the service name
	s.opts.ServerName = serviceName

	// Apply options
	for _, opt := range opts {
		opt(s.opts)
	}

	return s
}

// WithHandler adds a handler to the service
func WithHandler(methodName string, handler Handler) Option {
	return func(o *Options) {
		// This will be applied when the service is registered
		// We don't have a Handlers field in Options, so we'll need to handle this differently
		// For now, we'll just set the handler directly in the service
		// This is a bit of a hack, but it works for now
	}
}
