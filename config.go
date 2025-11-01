package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Port string `json:"port"`
	Host string `json:"host"`
}

func GetConfig() Config {
	configFile := "config.json"
	var config Config

	// 尝试打开配置文件
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) { // 配置文件不存在,创建默认配置文件

			defaultConfig := Config{
				Port: "7891",
				Host: "127.0.0.1",
			}

			file, err = os.Create(configFile)
			if err != nil {
				log.Fatalf("创建配置文件出错,错误信息:%v", err)
			}

			defer file.Close()

			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "  ")         // 设置缩进，第二个参数是缩进大小，这里使用两个空格
			err = encoder.Encode(defaultConfig) //写入默认配置
			if err != nil {
				log.Fatalf("错误编码文件(不可能发生),错误信息:%v", err)
			}
			fmt.Println("写入默认配置文件:", configFile)

			config = defaultConfig

		} else {
			log.Fatalf("打开配置文件失败,错误信息:%v", err)
		}
	} else {
		defer file.Close()

		// 读取现有配置
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			log.Fatalf("配置文件已经损坏,请检查或删除配置文件,错误信息:%v", err)
		}
	}
	// 输出当前配置
	return config
}
