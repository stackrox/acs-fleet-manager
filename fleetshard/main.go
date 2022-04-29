package main

import (
	"flag"
	"github.com/golang/glog"
	"time"
)

func play(sender chan<- int, receiver <-chan int, text string, sleepDuration time.Duration) {
	for {
		<-receiver
		glog.Info(text)
		time.Sleep(sleepDuration)
		sender <- 1
	}
}

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	ping := make(chan int)
	pong := make(chan int)

	go play(ping, pong, "ping", 1*time.Second)
	go play(pong, ping, "pong", 2*time.Second)

	glog.Info("fleetshard application has been started")

	ping <- 1

	for {
		time.Sleep(time.Second)
	}
}
