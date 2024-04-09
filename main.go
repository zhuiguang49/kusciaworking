package main

import(
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"log"
	"github.com:zhuiguang49/kusciaworking/metricexporter"
)
func main(){
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	//初始化service的URL
	metricURLs :=map[string]string{
		"service1":"http://service1.example.com/metrics",
		"service2":"http://service2.example.com/metrics",
	}

	//启动 MetricExporter
	go metricexporter.MetricExporter(ctx, metricURLs, "9094")
	log.Println("MetricExporter has been started on port 9094.")

	//等待中断信号以优雅地关闭服务
	<-ctx.Done()
	log.Println("Shutting down MetricExporter...")
}
