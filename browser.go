package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser 用于在默认浏览器中打开指定的URL
func OpenBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "darwin": // macOS
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		err = fmt.Errorf("不支持的操作系统")
	}
	return err
}
