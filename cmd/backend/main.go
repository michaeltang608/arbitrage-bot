package main

import (
	"github.com/urfave/cli"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"ws-quant/pkg/e"
	logger "ws-quant/pkg/log"
	"ws-quant/server/backend"
)

const VERSION = "v1.0.0"
const NAME = "ws-quant"

var log = logger.NewLog("binan")

func main() {
	defer e.Recover()()
	runtime.GOMAXPROCS(runtime.NumCPU())

	app := cli.NewApp()
	app.Name = NAME
	app.Version = VERSION
	app.Action = Start

	err := app.Run(os.Args)
	if err != nil {
		log.Panic("app start fail: ", err)
	}
}

func Start(cxt *cli.Context) error {
	svr := backend.New()

	sigCh := make(chan os.Signal)
	defer close(sigCh)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			// 收到停止信号，通知关闭服务
			println("收到停止信号了")
			time.Sleep(time.Second)
			_ = svr.QuantClose()
		}
	}()
	err := svr.QuantRun()
	if err != nil {
		log.Panic("quant run fail", err.Error())
	}

	return nil
}
