package server

import (
	"fmt"
)

type Server struct {
	 // key 是服务名，v是具体的服务
	services map[string]Service 
}

func NewServer() *Server {
	// 1. 创建一个Server实例
	server := &Server{
		services : make(map[string]Service),
	}
	return server
}

func (s *Server)Register(serviceDesc *ServiceDesc, svr interface{}) error{
	// 这里会注册到所有的service中，每一个service都有完整的方法路由（包括所有service的）
	for _, service := range s.services {
		err := service.Register(serviceDesc, svr)
		if err != nil {
			return fmt.Errorf("register service %s failed: %v", serviceDesc.ServiceName, err)
		}
	}
	return nil
}

func (s *Server)Serve(address string) error {
	
	// 调用每一个service的Serve方法
	for _, service := range s.services {
		service.Serve(address)
	}
	
	return nil	
}

