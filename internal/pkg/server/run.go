package server

import (
	"database/sql"
	"context"
	v1 "go-grpc/api/server/v1"
	service "go-grpc/internal/service/server/v1"
	// swagger "go-grpc/internal/pkg/swagger"
	// "github.com/elazarl/go-bindata-assetfs"
	"go-grpc/internal/pkg/util"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	
	"golang.org/x/sync/errgroup"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"crypto/tls"
	"path/filepath"
)

var cfg *Config

// 直接运行整个server，运行server的时候，要负责控制所有东西的生命周期
func RunServer() error {
	// 首先加载配置文件
	var err error
	cfg, err = newConfig()
	if err != nil {
		return fmt.Errorf("读取配置文件失败：%v", err)
	}

	// 创建http server的listen
	gListen, err := net.Listen("tcp", cfg.Server.Proxy)
	if err != nil {
		return fmt.Errorf("错误的TCP端口配置：%v", err)
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
	// 创建一个server stub，等下注册到grpc server中，因为强依赖了一个DB，所以要在这一层cancel的时候把它close掉
	v1API := service.NewToDoServiceServer(db)
	
	// 创建context
	ctx, cancel := context.WithCancel(context.Background())
	// 这里尝试使用errgroup，对os.signal和ctx的管道进行监听，并且开启服务
	g, ctx := errgroup.WithContext(ctx)
	
	// 创建通用型server，如果开启了TLS，那么grpc+gateway都会在这个server
	var server *http.Server

	// 创建grpc server，如果没有开启TLS的话，grpc和gateway必须要分开，因为同一个端口监听会造成grpc偶尔失效的情况
	var grpcServer *grpc.Server

	// tls的config，开启TLS的话，这个指针就会被初始化
	var tlsConfig *tls.Config

	if !cfg.Server.TLS.Enabled {  // 如果没有开启TLS，则直接用grpc和gateway分离的方式
		listen, err := net.Listen("tcp", cfg.Server.Host)
		if err != nil {
			cancel()
			return fmt.Errorf("错误的server端口配置：%v", err)
		}
		// 如果没有开启TLS，grpcServer一般不会报错
		grpcServer, _ = newGrpcServer(v1API, false)
		g.Go(func() error {
			return grpcServer.Serve(listen)
		})
		log.Printf("GRPC服务开启监听，服务Host：%s\n", cfg.Server.Host)
		// 创建gateway的server，没有grpc
		server = newServer(ctx, nil, nil)
	} else {
		// 开启了TLS，则首先初始化tls的config
		tlsConfig, err = util.GetTLSConfig(cfg.Server.TLS.CertPemPath, cfg.Server.TLS.CertKeyPath)
		if err != nil {
			cancel()
			return fmt.Errorf("读取TLS文件失败：%v", err)
		}
		// 然后创建一个通用的server，包含grpc和gateway
		server = newServer(ctx, tlsConfig, v1API)
	}

	// 开启server服务监听，这是一个HTTP的server，如果开启了TLS，它可以整合grpc和HTTP的监听，否则只能作为grpc的gateway
	g.Go(func () error {
		if cfg.Server.TLS.Enabled {
			return server.Serve(tls.NewListener(gListen, tlsConfig))
		} else {
			return server.Serve(gListen)
		}
		
	})
	log.Printf("服务开启监听，服务Host：%s\n", cfg.Server.Proxy)
	

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
				if !cfg.Server.TLS.Enabled {
					grpcServer.GracefulStop()
				}
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

// 创建http服务的server，如果有tls.Config，则连同grpc一起创建
func newServer(ctx context.Context, tlsConfig *tls.Config, v1API v1.ToDoServiceServer) *http.Server {
		// 创建gateway的mux
	gmux, err := newGateway(ctx)
	if err != nil {
		panic(err)
	}

	// 创建一个http的mux，然后把gmux、swagger的handler都注册进去
	mux := http.NewServeMux()
	mux.Handle("/", gmux)
	mux.HandleFunc("/swagger/", SwaggerFileFunc)
	registerSwaggerUI(mux)

	// 先初始化一个handler
	var handler http.Handler
	// 创建grpc server
	var grpcServer *grpc.Server
	if tlsConfig != nil {
		grpcServer, err = newGrpcServer(v1API, true)
		handler = GrpcHandlerFunc(grpcServer, mux)
	} else {
		handler = mux
	}

	// 创建一个http.Server，并返回
	return &http.Server {
		Addr: cfg.Server.Host,
		Handler: handler,
	}
}

// 创建一个GRPC的server
func newGrpcServer(v1API v1.ToDoServiceServer, tls bool) (*grpc.Server, error) {
	// grpc的选项，根据有没有开启TLS来创建
	var opts []grpc.ServerOption
	if tls {
		creds, err := credentials.NewServerTLSFromFile(cfg.Server.TLS.CertPemPath, cfg.Server.TLS.CertKeyPath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(creds))
	}
	// 向grpc注册server stub
	grpcServer := grpc.NewServer(opts...)
	v1.RegisterToDoServiceServer(grpcServer, v1API)
	return grpcServer, nil
}

//  根据官方的提示，创建一个gateway的mux
func newGateway(ctx context.Context) (http.Handler, error) {
	var endpoint string
	var opts []grpc.DialOption
	// 如果有开启TLS，则gateway和grpc是处于同一个proxy端口的
	if cfg.Server.TLS.Enabled {
		endpoint = cfg.Server.Proxy
		dcreds, err := credentials.NewClientTLSFromFile(cfg.Server.TLS.CertPemPath, cfg.Server.TLS.CommonName)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(dcreds))
	} else {
		endpoint = cfg.Server.Host
		opts = append(opts, grpc.WithInsecure())
	}
	mux := runtime.NewServeMux()
	err := v1.RegisterToDoServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
	if err != nil {
		return nil, err
	}
	return mux, nil
}

// 将swagger的文件注册到mux中
func registerSwaggerUI(mux *http.ServeMux) {
	// 首先swagger的目录是固定的，但是调用的路径不一定是固定的，所以一定要从相对路径找到绝对路径
	prefix := "/swagger-ui/"
	swaggerDir := http.Dir(filepath.Join(BaseDir, "../../internal/pkg/third_party/swagger-ui"))
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir(swaggerDir))))
}
