package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/julienschmidt/httprouter"
	pshandler "github.com/luqmanahmads/chatapp/internal/handler/pubsub"
)

func main() {
	log.SetFlags(log.LstdFlags)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	s := &http.Server{
		Handler:      setupHandler(),
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		Addr:         ":8008",
	}

	errc := make(chan error, 1)
	go func() {
		errc <- s.ListenAndServe()
	}()
	log.Printf("application started at :8008")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to listen and serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}

func setupHandler() http.Handler {
	pubsubHandler := pshandler.New()

	router := httprouter.New()
	router.GET("/", pubsubHandler.HandleWelcome)
	router.POST("/publish", pubsubHandler.HandlePublish)
	router.GET("/subscribe", pubsubHandler.HandleSubscribe)

	return router
}
