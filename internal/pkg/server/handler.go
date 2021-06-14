package server

import (
	"net/http"
	"strings"
	"log"
	"google.golang.org/grpc"
	"path"
)

func GrpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	if otherHandler == nil {
		return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			grpcServer.ServeHTTP(w, r)
		})
	}

	// 这是一个兼容grpc请求和http请求的handler，可以根据请求类型判断，从而选择某个handler去完成这个请求
	// 但是如果使用grpc Server的serveHTTP方法的话，要求必须要有TLS协议，也就是HTTPS，所以这里HTTP是不行的
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && len(r.Header) == 0 {
			log.Printf("RPC Request: %s %s from %s\n", r.Method, r.RequestURI, r.RemoteAddr)
			grpcServer.ServeHTTP(w, r)
		} else {
			// 输出请求的详情，比如HTTP Request: Method URI from RemoteAddr
			log.Printf("%s Request: %s %s from %s\n", r.Proto, r.Method, r.RequestURI, r.RemoteAddr)
			otherHandler.ServeHTTP(w, r)
		}
	})
}

func SwaggerFileFunc(w http.ResponseWriter, r *http.Request) {
	if ! strings.HasSuffix(r.URL.Path, "swagger.json") {
        log.Printf("Not Found: %s", r.URL.Path)
        http.NotFound(w, r)
        return
    }

    p := strings.TrimPrefix(r.URL.Path, "/swagger/")
    p = path.Join("../../api/server/v1", p)

    log.Printf("Serving swagger-file: %s", p)

    http.ServeFile(w, r, p)
}