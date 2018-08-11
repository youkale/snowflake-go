package main

import (
	"github.com/youkale/snowflake-go/app/serve"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	listen := *flag.String("listen", "localhost:8199", "listen address like localhost:8199")
	log.Printf(`snowflake listen on %s`, listen)
	s := serve.NewServe(listen)

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)

	s.Start()

	<-osSignals
	s.Close()
}
