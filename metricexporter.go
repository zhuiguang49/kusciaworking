// Copyright 2023 Ant Group Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metricexporter

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
	
	"github.com/zhuiguang49/kuscia/pkg/utils/nlog"
)

var (
	ReadyChan = make(chan struct{})
)

func getMetrics(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		nlog.Error("Error creating request:", err)
		return nil, err
	}
	client := http.Client{
		Timeout: 100 * time.Millisecond,
	}
	response, err := client.Do(request)
	if err != nil {
		nlog.Error("Error sending request:", err)
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		nlog.Error("Error reading response body:", err)
		return nil, err
	}
	return responseBody, nil
}
func metricHandler(metricUrls map[string]string, w http.ResponseWriter) {
	metricsChan := make(chan []byte, len(metricUrls))
	var wg sync.WaitGroup

	for key, url := range metricUrls {
		wg.Add(1)
		go func(key string, url string) {
			defer wg.Done()

			metrics, err := getMetrics(url)
			if err == nil {
				metricsChan <- metrics
			} else {
				nlog.Warnf("metrics[%s] query failed", key)
				metricsChan <- nil // empty metrics
			}
		}(key, url)
	}

	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	for metrics := range metricsChan {
		if metrics != nil {
			w.Write(metrics)
		}
	}
}

func MetricExporter(ctx context.Context, metricURLs map[string]string, port string) {
	nlog.Info("Start to export metrics...")
	metricServer := http.NewServeMux()
	metricServer.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricHandler(metricURLs, w)
	})
	server := &http.Server{
		Addr:    ":" + port,
		Handler: metricServer,  // 使用 metricServer 处理请求
	}

	go func() {
		//服务器将在所有可用网络接口的该端口上监听请求
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			nlog.Error("Error starting HTTP server:", err)
		}
	}()
	defer func() {
		close(ReadyChan)
		nlog.Info("Metric exporter shut down gracefully.")
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//关闭HTTP服务器
	if err := server.Shutdown(shutdownCtx); err != nil {
		nlog.Error("HTTP server shutdown error:", err)
	} else {
		nlog.Info("Metric exporter shut down gracefully.")
	}
}
