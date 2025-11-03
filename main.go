package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {

	// Module_webdav()
	// Module_open()

	Module_proxy()
	http_listen()

}

func http_listen() {
	config := GetConfig()
	address := config.Host + ":" + config.Port
	fmt.Println("监听地址：", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
