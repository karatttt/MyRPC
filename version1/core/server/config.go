package server

type ServerConfig struct {
	Server struct {
		Service []struct {
			Name string `yaml:"name"`
			IP   string `yaml:"ip"`
			Port int    `yaml:"port"`
		} `yaml:"service"`
	} `yaml:"server"`
}
