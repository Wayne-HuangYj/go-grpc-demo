package swagger

import (
	"strings"
	"path"
	"net/http"
	"log"
	// "github.com/elazarl/go-bindata-assetfs"
)

func ServeSwaggerFile(w http.ResponseWriter, r *http.Request) {
	// 判断有无指定对应的后缀
	if ! strings.HasSuffix(r.URL.Path, "swagger.json") {
		log.Printf("Not Found: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	// 在URL中，去掉/swagger/的前缀，获取剩下的后缀，TrimPrefix如果不是以这个前缀开头，则返回字符串本身
	p := strings.TrimPrefix(r.URL.Path, "/swagger/")
	p = path.Join("../../api/server/v1/", p)

	log.Printf("Serving swagger-file: %s", p)

	http.ServeFile(w, r, p)
}

// 对xxx.swagger.json提供文件访问支持，利用的是third_party里面的dist文件
func ServeSwaggerUI(mux *http.ServeMux) {
	
	// fileServer := http.FileServer(&assetfs.AssetFS{
	// 	// 将datafile.go中的函数作为这个AssetFS结构体的参数，虽然暂时还不知道有什么用
	// 	Asset: Asset,
	// 	AssetDir: AssetDir,
	// 	Prefix: "",
	// })
	prefix := "/swagger-ui/"
	// mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir("../../internal/pkg/third_party/swagger-ui"))))
}