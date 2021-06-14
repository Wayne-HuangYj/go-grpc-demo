package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"flag"
	// "os"
	// "fmt"
	// "os/exec"
	// "path/filepath"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Proxy string `yaml:"proxy"`
	}
	Mysql struct {
		Host string `yaml:"host"`
		User string `yaml:"user"`
		Password string `yaml:"password"`
		DBSchema string `yaml:"db-schema"`
	}
}

var cfg Config

func NewConfig() (*Config, error) {
	// file, _ := exec.LookPath(os.Args[0])
	// path, _ := filepath.Abs(file)
	// fmt.Println(path, string(os.PathSeparator))
	yamlFile, err := ioutil.ReadFile("../../configs/server/config.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, err
	}
	flag.StringVar(&cfg.Server.Host, "endpoint", cfg.Server.Host, "grpc port to bind")
	flag.StringVar(&cfg.Server.Proxy, "Grpc gateway", cfg.Server.Proxy, "grpc gateway port for http to bind")
	flag.StringVar(&cfg.Mysql.Host, "db-host",  cfg.Mysql.Host, "db host")
	flag.StringVar(&cfg.Mysql.User, "db-user",  cfg.Mysql.User, "db user")
	flag.StringVar(&cfg.Mysql.Password, "db-password", cfg.Mysql.Password, "db password")
	flag.StringVar(&cfg.Mysql.DBSchema, "db-schema", cfg.Mysql.DBSchema, "db schema")
	flag.Parse()
	return &cfg, nil
}
