package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/julienschmidt/httprouter"
	pshandler "github.com/luqmanahmads/chatapp/internal/handler/http"
	chatrepo "github.com/luqmanahmads/chatapp/internal/repository/chat"
	chatuc "github.com/luqmanahmads/chatapp/internal/usecase/chat"
	"github.com/nsqio/go-nsq"
)

func main() {
	log.SetFlags(log.LstdFlags)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Start HTTP server.
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

	// Shutdown HTTP Server.
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
	// Init Components.
	nsqProducer, err := nsq.NewProducer("localhost:4150", nsq.NewConfig())
	if err != nil {
		panic(err.Error())
	}

	// Init Repos.
	chatRepo := chatrepo.New(nsqProducer)

	// Init Usecases.
	chatUC := chatuc.New(chatRepo)

	// Init Handlers.
	pubsubHandler := pshandler.New(chatUC)

	router := httprouter.New()
	router.GET("/", pubsubHandler.HandleWelcome)
	router.POST("/publish", pubsubHandler.HandlePublish)
	router.GET("/subscribe", pubsubHandler.HandleSubscribe)

	router.POST("/v2/publish", pubsubHandler.HandlePublishV2)
	router.GET("/v2/subscribe", pubsubHandler.HandleSubscribeV2)

	return router
}
