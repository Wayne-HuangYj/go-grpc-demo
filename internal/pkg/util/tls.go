package util

import (
	"crypto/tls"
	"io/ioutil"
	"golang.org/x/net/http2"
)
// 获取TLS配置，读取server.key和server.pem
func GetTLSConfig(certPemPath, certKeyPath string) (*tls.Config, error) {
	// 保存公钥/私钥对
	var certKeyPair *tls.Certificate
	// 先读取两个文件
	cert, err := ioutil.ReadFile(certPemPath)
	if err != nil {
		return nil, err
	}
	key, err := ioutil.ReadFile(certKeyPath)
	if err != nil {
		return nil, err
	}
	// 从PEM编码中解析出公钥/私钥对
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	
	certKeyPair = &pair
	//  NextProtoTLS是谈判期间的NPN/ALPN协议，用于HTTP/2的TLS设置
	return &tls.Config {
		Certificates: []tls.Certificate{*certKeyPair},
		NextProtos: []string{http2.NextProtoTLS},
	}, nil
}