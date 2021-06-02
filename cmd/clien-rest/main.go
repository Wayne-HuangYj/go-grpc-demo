package main

import (
	// "gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"fmt"
	"flag"
	"time"
	"net/http"
	"strings"
	"encoding/json"
)

type config struct {
	Server struct {
		Host string `yaml:"host"`
	}
}


func main() {
	// yamlFile, err := ioutil.ReadFile("../../configs/client/config.yaml")
	// var cfg config
	// if err != nil {
	// 	log.Fatal("配置文件失效，请检查")
	// }
	// err = yaml.Unmarshal(yamlFile, &cfg)
	// if err != nil {
	// 	log.Fatal("配置文件解析失败，请检查")
	// }
	// address := flag.String("server", fmt.Sprintf("http://%s", cfg.Server.Host), "HTTP网关URL，e.g. http://localhost:" + cfg.Server.Host)
	address := flag.String("server", fmt.Sprintf("http://%s", "localhost:8080"), "HTTP网关URL，e.g. http://localhost:8080")
	flag.Parse()

	t := time.Now().In(time.UTC)
	pfx := t.Format(time.RFC3339Nano)

	var body string

	resp, err := http.Post(*address + "/v1/todo",
			"application/json",
			strings.NewReader(fmt.Sprintf(`
			{
					"api" : "v1",
					"toDo": {
						"title": "title (%s)",
						"description": "description (%s)",
						"reminder": "%s"
					}
				}
	`, pfx, pfx, pfx)))
	if err != nil {
		log.Fatalf("Create失败：%v", err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		body = fmt.Sprintf("Create response body 读取失败：%v", err)
	} else {
		body = string(bodyBytes)
	}
	log.Printf("Create response: Code=%d, Body=%s\n\n", resp.StatusCode, body)

	// 反序列化这个json string
	var created struct {
		API string `json:"api"`
		ID string `json:"id"`
	}

	err = json.Unmarshal(bodyBytes, &created)
	if err != nil {
		log.Fatalf("JSON response of Create method 反序列化失败：%v", err)
	}

	fmt.Println(created)
}