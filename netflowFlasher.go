package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	bufferSize     = 32 * 1024
	retryDelay     = 3 * time.Second
	requestTimeout = 10 * time.Second
	headerTimeout  = 15 * time.Second
)

type Config struct {
	DownloadList  []string `json:"downloadList"`
	MaxSpeedMiBps float64  `json:"maxSpeedMiBps"`
	EnableLog     bool     `json:"enableLog"`
}

func readConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	cfg := &Config{}
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件: %w", err)
	}

	// Reject trailing JSON values or other non-whitespace content.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("配置文件包含多余的 JSON 内容")
		}
		return nil, fmt.Errorf("配置文件尾部内容无效: %w", err)
	}

	if len(cfg.DownloadList) == 0 {
		return nil, errors.New("downloadList 不能为空")
	}
	for i, raw := range cfg.DownloadList {
		u, err := url.ParseRequestURI(strings.TrimSpace(raw))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, fmt.Errorf("downloadList[%d] 不是有效的 HTTP/HTTPS URL", i)
		}
		cfg.DownloadList[i] = strings.TrimSpace(raw)
	}
	if math.IsNaN(cfg.MaxSpeedMiBps) || math.IsInf(cfg.MaxSpeedMiBps, 0) || cfg.MaxSpeedMiBps <= 0 {
		return nil, errors.New("maxSpeedMiBps 必须是大于 0 的有限数值")
	}
	// Keep duration conversion and resource use within sensible bounds.
	if cfg.MaxSpeedMiBps > 1024 {
		return nil, errors.New("maxSpeedMiBps 不能大于 1024")
	}
	return cfg, nil
}

func newHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   requestTimeout,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          2,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   requestTimeout,
		ResponseHeaderTimeout: headerTimeout,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{Transport: transport}
}

// consume reads fixed-size chunks and discards them. It deliberately waits
// after every chunk instead of accumulating unused bandwidth credit: if the
// server or network stalls, the program never "catches up" with a burst.
func consume(resp *http.Response, bytesPerSecond float64) (int64, error) {
	defer resp.Body.Close()

	buf := make([]byte, bufferSize)
	var total int64

	for {
		n, err := io.ReadFull(resp.Body, buf)
		if n > 0 {
			total += int64(n)
			wait := time.Duration(float64(n) / bytesPerSecond * float64(time.Second))
			if wait > 0 {
				time.Sleep(wait)
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func main() {
	cfg, err := readConfig("config.json")
	if err != nil {
		log.Fatalf("配置错误: %v", err)
	}

	if !cfg.EnableLog {
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}

	client := newHTTPClient()
	defer client.CloseIdleConnections()

	bytesPerSecond := cfg.MaxSpeedMiBps * 1024 * 1024
	var totalDownloaded int64

	for {
		for _, target := range cfg.DownloadList {
			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
			if err != nil {
				log.Printf("创建请求失败: %v", err)
				time.Sleep(retryDelay)
				continue
			}
			req.Header.Set("User-Agent", "netflowFlasher")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("请求失败 (%s): %v", target, err)
				time.Sleep(retryDelay)
				continue
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				status := resp.Status
				io.Copy(io.Discard, io.LimitReader(resp.Body, 4*1024))
				resp.Body.Close()
				log.Printf("HTTP 状态异常 (%s): %s", target, status)
				time.Sleep(retryDelay)
				continue
			}

			n, err := consume(resp, bytesPerSecond)
			totalDownloaded += n
			if err != nil {
				log.Printf("读取中断 (%s): %v；本次读取 %s", target, err, formatBytes(n))
				time.Sleep(retryDelay)
				continue
			}
			log.Printf("资源结束 (%s)，本次读取 %s，累计读取 %s", target, formatBytes(n), formatBytes(totalDownloaded))
		}
	}
}

func formatBytes(b int64) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
		TiB = 1024 * GiB
	)
	switch {
	case b >= TiB:
		return fmt.Sprintf("%.2f TiB", float64(b)/float64(TiB))
	case b >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(GiB))
	case b >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(b)/float64(MiB))
	case b >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(b)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
