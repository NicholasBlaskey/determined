package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

var configFile = flag.String("config-file", "", "config file")

func TestMainFunc(t *testing.T) {
	done := make(chan os.Signal)
	log.Println("Starting server")
	go main()
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Waiting for exit")
	<-done
	log.Println("Exiting")
	return
}
