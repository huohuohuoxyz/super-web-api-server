package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func Module_proxy() {
	fmt.Println("启用代理服务")
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

		newRequest, err := http.NewRequest(r.Method, targetURLStr, r.Body)
		if err != nil {
			http.Error(w, "Failed to create new request", http.StatusInternalServerError)
			return
		}

		const huoHeaderPrefix = "huo-"
		var skipHeaders = map[string]bool{
			"host":       true,
			"origin":     true,
			"referer":    true,
			"connection": true,
			"upgrade":    true,
			// "content-length": true, //前面设置了conteng-length,如果再在header中设置,可能会出问题
			"sec-fetch-dest": true,
			"sec-fetch-site": true,
		}

		// 复制header
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

		// 处理我的自定义header,huo-*
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

		//调试打印出请求发送的header
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

		// 将响应header返回给客户端
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
