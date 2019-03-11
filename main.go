package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s, err := NewServer("127.0.0.1:1111")
	if err != nil {
		log.Println(err)
	}
	errCh := make(chan error)
	go func() {
		errCh <- s.Run()
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)
		log.Println("Got signal: ", <-c)
		errCh <- nil
	}()

	if err := <-errCh; err != nil {
		log.Println(err)
	}
	if s.Shutdown() != nil {
		log.Println("Error during shutdown", err)
	}
}
