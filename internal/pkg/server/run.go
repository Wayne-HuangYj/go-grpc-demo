package server

import (
	"database/sql"
	"context"
	v1 "go-grpc/api/server/v1"
	service "go-grpc/internal/service/server/v1"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"google.golang.org/grpc"
	config "go-grpc/configs/server"
	"golang.org/x/sync/errgroup"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

var cfg *config.Config

// 直接运行整个server，运行server的时候，要负责控制所有东西的生命周期
func RunServer() error {
	// 首先加载配置文件
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		return err
	}
	fmt.Println(&cfg)
	// 创建listen
	listen, err := net.Listen("tcp", cfg.Server.Host)
	gListen, err := net.Listen("tcp", cfg.Server.Proxy)
	if err != nil {
		return fmt.Errorf("错误的server端口配置：%v", err)
	}
	// 连接数据库，数据库实例是用于创建server stub的，实际上这种设计明显不好，直接将DAO放在service
	// 实际上service应该依赖于DAO的抽象接口，而DAO下面可以实现依赖倒置
	param := "parseTime=true"
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", 
								cfg.Mysql.User, cfg.Mysql.Password, cfg.Mysql.Host, cfg.Mysql.DBSchema, param)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}
	// 创建一个server stub，等下注册到grpc server中，因为强依赖了一个DB，所以要在这一层把它close掉
	v1API := service.NewToDoServiceServer(db)

	
	// 创建context
	ctx, cancel := context.WithCancel(context.Background())
	// 创建server，包含的是http的gateway和swagger的server
	server := newServer(ctx, v1API)
	// 创建grpc server，如果使用HTTP的话，grpc和gateway必须要分开，因为同一个端口监听会造成grpc偶尔失效的情况
	grpcServer := newGrpcServer(v1API)
	
	
	// 对os.signal和ctx的管道进行监听，这里尝试使用errgroup
	g, ctx := errgroup.WithContext(ctx)
	// 开启HTTP服务监听
	g.Go(func () error {
		return server.Serve(gListen)
	})
	log.Printf("Proxy服务开启监听，服务Host：%s\n", cfg.Server.Proxy)

	// 开启grpc监听
	g.Go(func() error {
		return grpcServer.Serve(listen)
	})
	log.Printf("GRPC服务开启监听，服务Host：%s\n", cfg.Server.Host)

	// 创建信号监听
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan)
	// 监听ctx.Done和signal
	g.Go(func() error {
		for {
			select {
			case s := <-signalChan:
				// 调用cancel，关闭ctx.Done管道，让所有goroutine都关闭
				cancel()
				log.Printf("接收到 \"%v\" 信号!准备关闭服务\n", s)
			case <-ctx.Done(): 
				// Done在调用cancel、timeout、deadlien后都会被close，所以把自己关掉
				server.Shutdown(ctx)
				grpcServer.GracefulStop()
				return ctx.Err()
			}
		}
	})

	if err := g.Wait(); err != nil {
		db.Close()
		log.Printf("服务退出，原因：%v\n", err)
	}
	return err
}

// 创建http服务的server
func newServer(ctx context.Context, v1API v1.ToDoServiceServer) *http.Server {
	// 创建grpc server
	grpcServer := newGrpcServer(v1API)
	// 创建gateway的mux
	gmux, err := newGateway()
	if err != nil {
		panic(err)
	}

	// 创建一个http的mux，然后把gmux、swagger的handler都注册进去
	mux := http.NewServeMux()
	mux.Handle("/", gmux)
	mux.HandleFunc("/swagger/", SwaggerFileFunc)
	registerSwaggerUI(mux)


	// 创建一个http.Server，并返回
	return &http.Server {
		Addr: cfg.Server.Host,
		Handler: GrpcHandlerFunc(grpcServer, mux),
	}
}

// 创建一个GRPC的server
func newGrpcServer(v1API v1.ToDoServiceServer) *grpc.Server {
	// 向grpc注册server stub
	grpcServer := grpc.NewServer()
	v1.RegisterToDoServiceServer(grpcServer, v1API)
	return grpcServer


	// 监听信号，有什么信号都退出处理
	// c := make(chan os.Signal, 1)
	// signal.Notify(c)
	// go func() {
	// 	select {
	// 	case signal := <-c:
	// 		log.Printf("shutting down gRPC server: %v signal received...\n", signal)
	// 		grpcServer.GracefulStop()
	// 		<-ctx.Done()
	// 	}
	// }()
	// log.Printf("starting gRPC server, listening on %s...\n", host)
	// // 开启grpc服务
	// return grpcServer.Serve(listen)
}

//  根据官方的提示，创建一个gateway的mux
func newGateway() (http.Handler, error) {
	ctx := context.Background()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	mux := runtime.NewServeMux()
	fmt.Println(&cfg)
	err := v1.RegisterToDoServiceHandlerFromEndpoint(ctx, mux, cfg.Server.Host, opts)
	if err != nil {
		return nil, err
	}
	return mux, nil
}

// 将swagger的文件注册到mux中
func registerSwaggerUI(mux *http.ServeMux) {
	prefix := "/swagger-ui/"
	// mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir("../../internal/pkg/third_party/swagger-ui"))))
}
