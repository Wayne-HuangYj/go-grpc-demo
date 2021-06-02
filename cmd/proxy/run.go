package main

import (
	v1 "go-grpc/api/server/v1"
	"google.golang.org/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"flag"
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v2"
	"context"
	"net/http"
	"fmt"
)

type config struct {
	Server struct {
		Host string `yaml:"host"`
	}
	Proxy struct {
		Host string `yaml:"host"`
	}
}

var cfg config

func init() {
	yamlFile, err := ioutil.ReadFile("../../configs/server/config.yaml")
	if err != nil {
		log.Fatalf("配置文件失效，请检查：%v\n", err)
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		log.Fatalf("配置文件解析失败，请检查：%v\n", err)
	}
	flag.StringVar(&cfg.Proxy.Host, "proxy-host", cfg.Proxy.Host, "gRPC proxy server host")
	flag.Parse()
}

func RunProxy() error {
	return runProxy()
}


func runProxy() error {
	// 按照官方的例子开启一个gateway
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
  	opts := []grpc.DialOption{grpc.WithInsecure()}
	// 注册的时候，要指定一个endpoint入口，这个入口是grpc的入口而不是gateway的入口
	err := v1.RegisterToDoServiceHandlerFromEndpoint(ctx, mux, cfg.Server.Host, opts)
	if err != nil {
		return err
	}
	log.Println("starting grpc-proxy server...")
	fmt.Println(cfg)
	return http.ListenAndServe(cfg.Proxy.Host, mux)
}