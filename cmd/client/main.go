package main

import (
	"context"
	"flag"
	v1 "go-grpc/api/server/v1"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"log"
	"fmt"
	"time"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

const (
	apiVersion="v1"
)

type config struct {
	Server struct {
		Host string `yaml:"host"`
	}
}

var cfg config

func init() {
	yamlFile, err := ioutil.ReadFile("../../configs/client/config.yaml")
	if err != nil {
		log.Fatal("配置文件失效，请检查")
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		log.Fatal("配置文件解析失败，请检查")
	}
}

func main() {
	address := flag.String("server", cfg.Server.Host, "gRPC server in format host:port")
	flag.Parse()
	conn, err := grpc.Dial(*address, grpc.WithInsecure())
	if err != nil {
		log.Fatal("服务器，连不上啊", err)
	}
	defer conn.Close()

	// 生成client stub
	c := v1.NewToDoServiceClient(conn)
	// 设置5秒ctx
	ctx := context.Background()

	// 生成时间戳
	t := time.Now().In(time.UTC)
	reminder, _ := ptypes.TimestampProto(t)
	pfx := t.Format(time.RFC3339Nano)

	req1 := v1.CreateRequest {
		Api: apiVersion,
		ToDo:&v1.ToDo {
			Title: "title (" + pfx + ")",
			Description: "description (" + pfx + ")",
			Reminder: reminder,
		},
	}
	// 调用RPC方法
	res1, err := c.Create(ctx, &req1)
	if err != nil {
		log.Fatal("创建失败:", err)
	}
	log.Printf("Create result: %v\n", res1)
	id := res1.Id

	req2 := v1.ReadRequest {Api: apiVersion, Id: id}
	res2, err := c.Read(ctx, &req2)
	if err != nil {
		log.Fatal("Read failed", err)
	}
	log.Printf("Read result %v\n", res2)

	req3 := v1.UpdateRequest {
		Api: apiVersion,
		ToDo: &v1.ToDo {
			Id: res2.ToDo.Id,
			Title: res2.ToDo.Title + "changed",
			Description: res2.ToDo.Description,
			Reminder: res2.ToDo.Reminder,
		},
	}
	res3, err := c.Update(ctx, &req3)
	if err != nil {
		log.Fatal("更新失败", err)
	}
	log.Printf("update result %v\n", res3)

	req4 := v1.ReadAllRequest {
		Api: apiVersion,
	}

	res4, err := c.ReadAll(ctx, &req4)
	if err != nil {
		log.Fatal("ReadAll 失败", err)
	}
	log.Printf("ReadAll result %v\n", res4)

	req5 := v1.DeleteRequest {
		Api: apiVersion,
		Id: id,
	}
	res5, err := c.Delete(ctx, &req5)
	if err != nil {
		log.Fatal("删除失败", err)
	}
	fmt.Printf("delete result %v\n", res5)
}