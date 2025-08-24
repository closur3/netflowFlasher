package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// Config 配置结构体
type Config struct {
	DownloadList []string `json:"downloadList"`
	DatachunkMB  int64    `json:"datachunkMB"` // 单位 MB
	Datachunk    int64    `json:"-"`           // 实际字节数
	Timelapse    int      `json:"timelapse"`   // 单位秒
}

// 读取配置文件
func readConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	config.Datachunk = config.DatachunkMB * 1024 * 1024
	return config, nil
}

// 设置日志格式（去掉默认时间戳）
func setupLogger() {
	log.SetFlags(0)
}

// 字节转 MB
func bytesToMB(b int64) int64 {
	return b / 1024 / 1024
}

func main() {
	setupLogger()

	config, err := readConfig("config.json")
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	var totalDownloaded int64
	round := 0

	for {
		round++
		log.Printf("[R%d] ================= 轮次开始 =================", round)

		for idx, url := range config.DownloadList {
			prefix := fmt.Sprintf("[R%d,DL%d]", round, idx+1)
			waitSec := rand.Intn(10)
			timeSleep := time.Duration(waitSec) * time.Second

			log.Printf("%s 准备下载（等待 %d 秒）", prefix, waitSec)
			time.Sleep(timeSleep)

			log.Printf("%s 正在下载: %s", prefix, url)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("%s 下载失败: %v", prefix, err)
				continue
			}

			func() {
				defer resp.Body.Close()

				var downloaded int64
				file := io.Discard

				ticker := time.NewTicker(time.Duration(config.Timelapse) * time.Second)
				defer ticker.Stop()

				for range ticker.C {
					n, err := io.CopyN(file, resp.Body, config.Datachunk)
					downloaded += n
					atomic.AddInt64(&totalDownloaded, n)

					if err != nil {
						if err == io.EOF {
							log.Printf("%s 下载完成，本次 %d MB", prefix, bytesToMB(downloaded))
						} else {
							log.Printf("%s 下载失败: %v", prefix, err)
						}
						break
					}
				}
			}()
		}

		log.Printf("[R%d] 本轮结束，累计下载 %d MB", round, bytesToMB(atomic.LoadInt64(&totalDownloaded)))
	}
}
