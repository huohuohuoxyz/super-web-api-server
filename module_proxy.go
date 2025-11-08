package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	fhttp "github.com/Danny-Dasilva/fhttp"
)

func Module_proxy() {
	fmt.Println("启动/proxy监听")

	//自定义chrome_ja3用来高度模拟chrome
	//chrome浏览器配置的JA3与user-agent
	const chrome_ja3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,23-0-11-18-51-65037-16-65281-45-13-27-5-10-17513-35-43,4588-29-23-24,0"
	const chrome_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {

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

		targetURLStr := r.URL.Query().Get("url")
		if strings.HasPrefix(targetURLStr, "https://dl-pc-zb-w.drive.quark") {
			fmt.Println("-----------------------------------------------------------------------000000000000000000000000000000000")

			client := &http.Client{}
			req, err := http.NewRequest("GET", targetURLStr, nil)
			if err != nil {
				fmt.Println("Error creating request:", err)
				return
			}

			req.Header.Set("User-Agent", header_userAgent)
			req.Header.Set("Referer", header_referer)
			req.Header.Set("Cookie", Cookie)
			req.Header.Set("Range", r.Header.Get("Range"))

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Error sending request:", err)
				return
			}
			defer resp.Body.Close()

			// 检查响应状态
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("下载请求失败: %v", err)
				}
				fmt.Println("响应内容:", string(bodyBytes))
				fmt.Printf("下载失败，状态码: %d", resp.StatusCode)
				return
			}

			for key, values := range resp.Header {
				for _, value := range values {
					fmt.Printf("<<<res Header: %s: %s\n", key, value)
					if strings.ToLower(key) == "set-cookie" {
						w.Header().Add("huo-header-set-cookie", value)
					} else {
						w.Header().Add(key, value)
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Expose-Headers", "*") // 允许前端读取所有响应头
			w.WriteHeader(resp.StatusCode)

			_, err = io.Copy(w, resp.Body)

			if err != nil {
				fmt.Printf("写入文件失败: %v", err)
			}

			return

			// targetURLStr = downurl
		}
		fmt.Println("==========有新请求=========\n"+targetURLStr, r.Method)
		if targetURLStr == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		// bodyBytes, err := io.ReadAll(r.Body)
		// fmt.Println("******请求体:", string(bodyBytes))
		// if err != nil {
		// 	http.Error(w, "Failed to read body: "+err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		bodyBytes, err := io.ReadAll(r.Body)
		fmt.Println("******请求体:", string(bodyBytes), "长度:", len(bodyBytes))
		if err != nil {
			http.Error(w, "Failed to read body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// 创建请求
		newRequest, err := fhttp.NewRequest(r.Method, targetURLStr, bytes.NewBuffer(bodyBytes))
		if err != nil {
			http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if bodyBytes != nil {
			fmt.Println("设置长度")
			newRequest.ContentLength = int64(len(bodyBytes))
		}

		contentLength := r.Header.Get("Content-Length")
		if contentLength != "" {
			length, err := strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				http.Error(w, "Invalid Content-Length header", http.StatusBadRequest)
				return
			}
			newRequest.ContentLength = length
			fmt.Println("从header中获取到content-length", length)
		}

		const headerPrefix = "huo-"

		// newRequest.Header.Set("cookie", r.Header.Get("huo-set-cookie"))

		var skipHeaders = map[string]bool{
			"host":            true,
			"origin":          true,
			"referer":         true,
			"connection":      true,
			"upgrade":         true,
			"content-length":  true, //前面设置了conteng-length,如果再在header中设置,可能会出问题
			"sec-fetch-dest":  true,
			"sec-fetch-site":  true,
			"accept-language": true,
			"sec-fetch-mode":  true,
			"accept":          true,
			"accept-encoding": true,
		}
		for key, values := range r.Header {
			lowerKey := strings.ToLower(key)
			if !strings.HasPrefix(lowerKey, headerPrefix) {
				if skipHeaders[lowerKey] {
					fmt.Printf("跳过header '%s'\n", lowerKey)
				} else {
					for _, value := range values {
						newRequest.Header.Add(key, value)
					}
				}
			}
		}

		// 遍历客户端请求头部,来处理 huo-*
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
			}
		}

		// 设置User-Agent等请求头
		newRequest.Header.Set("User-Agent", header_userAgent)
		newRequest.Header.Set("Referer", header_referer)
		newRequest.Header.Set("Cookie", Cookie)
		newRequest.Header.Set("Accept", "application/json, text/plain, */*")
		newRequest.Header.Set("Range", "bytes=0-10000000")

		for key, values := range newRequest.Header {
			for _, value := range values {
				fmt.Printf(">>>req Header: %s: %s\n", key, value)
			}
		}

		// userAgent := newRequest.Header.Get("User-Agent")

		// if userAgent == "" {
		// 	userAgent = chrome_USER_AGENT
		// }
		fmt.Println("---------------------------------------------------------")
		client := &fhttp.Client{
			// Transport: cycletls.NewTransport(chrome_ja3, userAgent),
			// Timeout:   30 * time.Second,
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

		fmt.Println("<<<状态码:", resp.StatusCode)
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Printf("<<<res Header: %s: %s\n", key, value)
				if strings.ToLower(key) == "set-cookie" {
					w.Header().Add("huo-header-set-cookie", value)
				} else {
					w.Header().Add(key, value)
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*") // 允许前端读取所有响应头

		w.WriteHeader(resp.StatusCode)
		fmt.Println("复制数据")
		io.Copy(w, resp.Body)
		fmt.Println("复制数据完成")

		fmt.Println("处理完成==================================")
	})

}
