package main

import (
	"database/sql"
	"context"
	"flag"
	v1 "go-grpc/api/server/v1"
	service "go-grpc/internal/service/server/v1"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"os"
	"google.golang.org/grpc"
)

type config struct {
	Server struct {
		Host string `yaml:"host"`
	}
	Mysql struct {
		Host string `yaml:"host"`
		User string `yaml:"user"`
		Password string `yaml:"password"`
		DBSchema string `yaml:"db-schema"`
	}
}

var cfg config

func init() {
	yamlFile, err := ioutil.ReadFile("../../configs/server/config.yaml")
	if err != nil {
		log.Fatal("配置文件失效，请检查")
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		log.Fatal("配置文件解析失败，请检查")
	}
	flag.StringVar(&cfg.Server.Host, "grpc-host", cfg.Server.Host, "gRPC host to bind")
	flag.StringVar(&cfg.Mysql.Host, "db-host",  cfg.Mysql.Host, "db host")
	flag.StringVar(&cfg.Mysql.User, "db-user",  cfg.Mysql.User, "db-user")
	flag.StringVar(&cfg.Mysql.Password, "db-password", cfg.Mysql.Password, "db-password")
	flag.StringVar(&cfg.Mysql.DBSchema, "db-schema", cfg.Mysql.DBSchema, "db-schema")
	flag.Parse()
	
}

func RunServer() error {
	ctx := context.Background()
	
	if len(cfg.Server.Host) == 0 {
		return fmt.Errorf("invalid TCP host for gRPC server: %s", cfg.Server.Host)
	}
	param := "parseTime=true"
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", cfg.Mysql.User, cfg.Mysql.Password, cfg.Mysql.Host, cfg.Mysql.DBSchema, param)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}
	defer db.Close()

	v1API := service.NewToDoServiceServer(db)
	return runServer(ctx, v1API, cfg.Server.Host)
}

func runServer(ctx context.Context, v1API v1.ToDoServiceServer, host string) error {
	listen, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	v1.RegisterToDoServiceServer(server, v1API)
	c := make(chan os.Signal, 1)
	go func() {
		for range c {
			log.Println("shutting down gRPC server...")
			server.GracefulStop()
			<-ctx.Done()
		}
	}()
	log.Println("starting gRPC server...")
	return server.Serve(listen)
}

