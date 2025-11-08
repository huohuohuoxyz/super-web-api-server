package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func Module_proxy2() {

	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			//预检请求
			w.Header().Set("Access-Control-Allow-Origin", "*")
			//如果前端请求中包含了自定义头字段（如 Authorization 或 X-Token），浏览器会要求服务器明确指定允许的 HTTP 方法，而不是使用通配符 *。
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		targetURLStr := r.URL.Query().Get("url")
		fmt.Println("========有新请求========\n", targetURLStr, r.Method)
		if targetURLStr == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		var body io.Reader = nil
		if r.Method == http.MethodPost {
			bodyBytes, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, "Failed to read body: "+err.Error(), http.StatusInternalServerError)
				return
			}
			body = bytes.NewBuffer(bodyBytes)
			fmt.Println("******请求体:", string(bodyBytes), "长度:", len(bodyBytes))
		}

		newRequest, err := http.NewRequest(r.Method, targetURLStr, body)
		if err != nil {
			http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
			return
		}

		newRequest.Header = make(http.Header)

		const huoHeaderPrefix = "huo-"

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
			if !strings.HasPrefix(lowerKey, huoHeaderPrefix) {
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
			if strings.HasPrefix(lowerKey, huoHeaderPrefix) {
				// 去掉 prefix，例如 huo-set-User-Agent → set-User-Agent
				prefixedPart := strings.TrimPrefix(lowerKey, huoHeaderPrefix)

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

		newRequest.Header.Set("User-Agent", header_userAgent)
		newRequest.Header.Set("Referer", header_referer)
		newRequest.Header.Set("Cookie", Cookie)
		if r.Header.Get("Range") != "" {
			oldRange := r.Header.Get("Range")

			if strings.HasPrefix(oldRange, "bytes=") {
				rangeValue := strings.TrimPrefix(oldRange, "bytes=")
				parts := strings.Split(rangeValue, "-")
				start := parts[0]
				const maxRangeSize = 100000000 //100MB
				if start == "" {
					start = "0"
				}
				//将start转换int64
				startInt, err := strconv.ParseInt(start, 10, 64)
				if err != nil {
					http.Error(w, "Invalid Range header: "+err.Error(), http.StatusBadRequest)
					return
				}
				newRequest.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startInt, startInt+maxRangeSize))

			}

			// newRequest.Header.Set("Range", "bytes=0-10"+"000000")
		}

		for key, values := range newRequest.Header {
			for _, value := range values {
				fmt.Printf(">>>req Header: %s: %s\n", key, value)
			}
		}

		client := &http.Client{}

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

		// if r.Header.Get("Range") != "" {
		// 	//下载文件
		// 	out, err := os.Create("D:/file.mp4")
		// 	if err != nil {
		// 		fmt.Printf("创建文件失败: %v\n", err)
		// 	}
		// 	defer out.Close()

		// 	// 将响应体写入文件
		// 	_, err = io.Copy(out, resp.Body)
		// 	if err != nil {
		// 		fmt.Printf("写入文件失败: %v\n", err)
		// 	}
		// 	fmt.Println("写入文件成功")
		// 	return

		// }

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

		_, copyErr := io.Copy(w, resp.Body)
		if copyErr != nil {
			fmt.Println("复制数据时出错:", copyErr)
		}

		fmt.Println("处理完成==================================")

	})

}
