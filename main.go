package main

import (
	"context"
	"daarul_mukhtarin/internal/config"
	"daarul_mukhtarin/internal/factory"
	httpdaarul_mukhtarin "daarul_mukhtarin/internal/http"
	middlewareEcho "daarul_mukhtarin/internal/middleware"
	db "daarul_mukhtarin/pkg/database"
	"daarul_mukhtarin/pkg/log"
	"daarul_mukhtarin/pkg/ngrok"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// @title daarul_mukhtarin
// @version 1.0.0
// @description This is a doc for daarul_mukhtarin.

func main() {
	config.Init()

	log.Init()

	db.Init()

	e := echo.New()

	f := factory.NewFactory()

	middlewareEcho.Init(e, f.DbRedis)

	httpdaarul_mukhtarin.Init(e, f)

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runNgrok := true
		addr := ""
		if runNgrok {
			listener := ngrok.Run()
			e.Listener = listener
			addr = "/"
		} else {
			addr = ":" + config.Get().App.Port
		}
		err := e.Start(addr)
		if err != nil {
			if err != http.ErrServerClosed {
				logrus.Fatal(err)
			}
		}
	}()

	<-ch

	logrus.Println("Shutting down server...")
	cancel()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	e.Shutdown(ctx2)
	logrus.Println("Server gracefully stopped")
}
