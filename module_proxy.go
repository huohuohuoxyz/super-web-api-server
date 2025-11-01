package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	fhttp "github.com/Danny-Dasilva/fhttp"
)

func Module_proxy() {
	fmt.Println("启动/proxy监听")

	//自定义chrome_ja3用来突破一些网站的安全措施
	//chrome浏览器配置的JA3与user-agent
	const chrome_ja3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,23-0-11-18-51-65037-16-65281-45-13-27-5-10-17513-35-43,4588-29-23-24,0"
	const chrome_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		targetURLStr := r.URL.Query().Get("url")

		if r.Method == "PUT" {
			fmt.Println("PUT请求")
		}

		fmt.Println("有新请求========="+targetURLStr, r.Method)

		// 设置CORS头部
		if r.Method == http.MethodOptions {
			//预检请求
			w.Header().Set("Access-Control-Allow-Origin", "*")
			//如果前端请求中包含了自定义头字段（如 Authorization 或 X-Token），浏览器会要求服务器明确指定允许的 HTTP 方法，而不是使用通配符 *。
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			// w.Header().Set("Access-Control-Max-Age", "86400") // 预检请求缓存一天,好像没啥效果
			// fmt.Println("预请求处理完成==================")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if targetURLStr == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		// 解析目标 URL
		targetURL, err := url.Parse(targetURLStr)
		if err != nil {
			http.Error(w, "Invalid URL parameter", http.StatusBadRequest)
			return
		}

		// 创建请求
		newRequest := &fhttp.Request{
			Method: r.Method,
			URL:    targetURL,
			Header: make(fhttp.Header),
			Body:   r.Body,
		}

		const headerPrefix = "huo-"
		// 遍历客户端请求头部
		for key, values := range r.Header {
			lowerKey := strings.ToLower(key)
			if strings.HasPrefix(lowerKey, headerPrefix) {
				// 去掉 prefix，例如 huo-set-User-Agent → set-User-Agent
				prefixedPart := strings.TrimPrefix(lowerKey, headerPrefix)

				// 尝试按第一个 '-' 分割出操作和字段名，例如 set-User-Agent → ["set", "User-Agent"]
				parts := strings.SplitN(prefixedPart, "-", 2)
				if len(parts) < 2 {
					// 如果没有指定操作和字段（如只有 huo-xxx），忽略或记录警告
					fmt.Printf("⚠️  Invalid huo-prefixed header format (expected 'huo-[操作]-[字段]'), got: %s\n", key)
					continue
				}

				operation := parts[0]   // 如 "set", "del", "add"
				headerField := parts[1] // 如 "User-Agent"
				for _, value := range values {
					switch operation {
					case "set":
						newRequest.Header.Set(headerField, value)
					case "del":
						newRequest.Header.Del(headerField)
					case "add":
						newRequest.Header.Add(headerField, value)
					default:
						// 未知操作，可以选择 Set 作为默认行为，或者忽略
						fmt.Printf("[HUO-HEADER] Unknown operation '%s' for header '%s', treating as SET\n", operation, headerField)
					}
				}
			} else if lowerKey != "content-length" {
				for _, value := range values {
					newRequest.Header.Add(key, value)
				}
			}
		}

		for key, values := range newRequest.Header {
			for _, value := range values {
				fmt.Printf("req Header: %s: %s\n", key, value)
			}
		}

		userAgent := r.Header.Get("User-Agent")
		fmt.Println("userAgent", userAgent)

		client := &fhttp.Client{
			Transport: cycletls.NewTransport(chrome_ja3, userAgent),
			Timeout:   30 * time.Second,
		}

		// 发送请求并获取响应
		resp, err := client.Do(newRequest)
		if err != nil {
			fmt.Println("请求出错了.................====================================")
			// 根据不同类型的错误返回不同的HTTP状态码
			if urlErr, ok := err.(*url.Error); ok {
				if urlErr.Timeout() {
					http.Error(w, "Request timeout", http.StatusGatewayTimeout)
					return
				} else if urlErr.Temporary() {
					http.Error(w, "Temporary network error", http.StatusServiceUnavailable)
					return
				}
			}
			http.Error(w, "Failed to fetch resource: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Printf("res Header: %s: %s\n", key, value)
				w.Header().Add(key, value)
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*") // 允许前端读取所有响应头

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)

		fmt.Println("处理完成==================================")
	})

}
