package server

import (
	"MyRPC/core/transport"
	"context"
	"fmt"
	"encoding/json"
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
	Handler   map[string]Handler
	Transport *transport.ServerTransport
}

type Handler func(req []byte) (rsp []byte, err error)

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
	s.Transport = &transport.ServerTransport{}

	// TODO 现在只做了绑定方法，service后面可能也要绑定，以及router如何理解
	s.registerMethods(serviceDesc.Methods, service)

	return nil
}

// 注册普通方法
func (s *service) registerMethods(methods []MethodDesc, serviceImpl interface{}) error {
	for _, method := range methods {
		if _, exists := s.Handler[method.MethodName]; exists {
			return fmt.Errorf("duplicate method name: %s", method.MethodName)
		}
		s.Handler[method.MethodName] = func(req []byte) (rsp []byte, err error) {
			if fn, ok := method.Func.(func(svr interface{}, req []byte) (rsp []byte, err error)); ok {
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
	s.Transport.ConnHandler = s
	err := s.Transport.ListenAndServe(context.Background(), "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return nil
}

// 实现Handler接口，后续需要根据请求的methodName，然后调用这个service的Handler
// 里面的逻辑是：匹配service的HandlerMap是否有对应的methodName，如果有，则调用里面的Handler
func (s *service) Handle(req []byte) (rsp []byte, err error) {
	
	// 获取请求的methodName
	request := &transport.Request{}
	err = json.Unmarshal(req, request)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	// 获取Handler执行结果
	response, err := s.Handler[request.Method](request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to handle request: %v", err)
	}
	return response, nil
}

