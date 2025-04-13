package server

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Server struct {
	 // key 是服务名，v是具体的服务
	services map[string]Service 
}

func NewServer() *Server {
	// 1. 创建一个Server实例
	server := &Server{
		services: make(map[string]Service),
	}

	// 2. 读取配置文件
	config, err := loadConfig("./rpc.yaml")
	if err != nil {
		fmt.Print("读取配置文件出错")
	}

	// 3. 创建服务
	for _, svc := range config.Server.Service {
		// 创建服务
		service := NewService(svc.Name, WithAddress(fmt.Sprintf("%s:%d", svc.IP, svc.Port)))

		// 添加到服务映射
		server.services[svc.Name] = service
	}

	return server
}

// loadConfig loads the server configuration from a YAML file
func loadConfig(configPath string) (*ServerConfig, error) {
	// 读取配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析YAML
	var config ServerConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
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

