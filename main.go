package main

import (
	"modhub/route"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	route.RouterApi()
	// 捕捉系统信号，以便优雅地关闭程序
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号以便优雅地退出
	<-sigChan
}
