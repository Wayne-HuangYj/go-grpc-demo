package server

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"flag"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Proxy string `yaml:"proxy"`
		TLS struct {
			Enabled bool `yaml:"enabled"`
			CertKeyPath string `yaml:"certKeyPath"`
			CertPemPath string `yaml:"certPemPath"`
			CommonName string `yaml:"commonName"`
		}
	}
	Mysql struct {
		Host string `yaml:"host"`
		User string `yaml:"user"`
		Password string `yaml:"password"`
		DBSchema string `yaml:"dbSchema"`
	}
}

var BaseDir string

func newConfig() (*Config, error) {
	var cfg Config
	// 处理相对路径的问题，首先获取命令行执行可执行文件的参数
	// LookPath是在环境变量Path下、或者是当前目录路径下寻找这个可执行文件的路径
	// 意思就是如果传入的可执行文件名有/，则直接从当前目录开始找，否则从path开始找。可能找到的是一个相对路径或者绝对路径
	// file, _ := exec.LookPath(os.Args[0])
	// Abs则是返回一个文件的绝对路径
	// path, _ := filepath.Abs(file)
	//  可以看出上面两个步骤明显是有重叠的部分，至于为什么需要LookPath，那是因为LookPath是一个寻找的作用，它可以保证可执行文件存在后才能执行
	// 但是这里由于百分百保证可执行文件存在，所以根本不需要LookPath，直接abs即可
	BaseDir, _ = filepath.Abs(os.Args[0])
	// 找到最后一个/的下标
	index := strings.LastIndex(BaseDir, string(os.PathSeparator))
	// 进行一个截取
	BaseDir = BaseDir[:index]
	// 最后根据具体的路径，进行一个相对路径的寻找，就可以得到正确的文件路径了
	yamlFile, err := ioutil.ReadFile(filepath.Join(BaseDir, "../../configs/server/config.yaml"))
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, err
	}
	// 处理一下TLS默认路径，使其变成一个绝对路径
	cfg.Server.TLS.CertPemPath = filepath.Join(BaseDir, "../../", cfg.Server.TLS.CertPemPath)
	cfg.Server.TLS.CertKeyPath = filepath.Join(BaseDir, "../../", cfg.Server.TLS.CertKeyPath)
	flag.StringVar(&cfg.Server.Host, "endpoint", cfg.Server.Host, "grpc port to bind")
	flag.StringVar(&cfg.Server.Proxy, "gateway", cfg.Server.Proxy, "grpc gateway port for http to bind")
	flag.BoolVar(&cfg.Server.TLS.Enabled, "tls-enabled", cfg.Server.TLS.Enabled, "open TLS")
	flag.StringVar(&cfg.Server.TLS.CertKeyPath, "tls-key-path", cfg.Server.TLS.CertKeyPath, "TLS Key File path")
	flag.StringVar(&cfg.Server.TLS.CertPemPath, "tls-pem-path", cfg.Server.TLS.CertPemPath, "TLS Pem File path")
	flag.StringVar(&cfg.Server.TLS.CommonName, "tls-common-name", cfg.Server.TLS.CommonName, "TLS Common Name")
	flag.StringVar(&cfg.Mysql.Host, "db-host",  cfg.Mysql.Host, "db host")
	flag.StringVar(&cfg.Mysql.User, "db-user",  cfg.Mysql.User, "db user")
	flag.StringVar(&cfg.Mysql.Password, "db-password", cfg.Mysql.Password, "db password")
	flag.StringVar(&cfg.Mysql.DBSchema, "db-schema", cfg.Mysql.DBSchema, "db schema")
	flag.Parse()
	
	return &cfg, nil
}

func fileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}