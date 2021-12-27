package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baybaraandrey/alive"
)

var source = flag.String("address", "0.0.0.0", "listen address.")
var proto = flag.String("proto", "udp", "'udp'|'icmp'. Setting to 'icmp' requires that it be run with super-user privileges.")
var configFile = flag.String("config", "./config.yaml", "path to config file.")

var pid int

func main() {
	flag.Parse()
	pid = os.Getpid()
	log := log.New(os.Stdout, "WATCHER : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(log); err != nil {
		log.Println("WATCHER: error:", err)
		os.Exit(1)
	}
}

var watchers []*alive.Watcher

func initWatchers(conf *alive.Config, log *log.Logger) error {
	var err error
	watchers = make([]*alive.Watcher, len(conf.Hosts))
	log.Printf("WATCHER : init : hosts : %d\n", len(conf.Hosts))
	log.Printf("WATCHER : packets listening on : %s\n", *source)

	for i := 0; i < len(conf.Hosts); i++ {
		host := conf.Hosts[i]
		watchers[i], err = alive.NewWatcher(host.Addr, log)
		if err != nil {
			return err
		}
		w := watchers[i]

		if *proto == "icmp" {
			w.SetPrivileged(true)
		} else if *proto == "udp" {
			w.SetPrivileged(false)
		} else {
			return errors.New("wrong `proto`")
		}

		w.SetSource(*source)
		w.SetSize(host.Size)
		w.SetTTL(host.TTL)

		deadline, err := time.ParseDuration(host.ReadTimeout)
		if err != nil {
			return err
		}
		interval, err := time.ParseDuration(host.Interval)
		if err != nil {
			return err
		}
		w.SetReadDeadline(deadline)
		w.SetInterval(interval)

		log.Printf("WATCHER : init : %s\n", w.Addr())
		log.Printf("WATCHER : \t   read-timeout : %v\n", deadline)
		log.Printf("WATCHER : \t   interval : %v\n", interval)
		log.Printf("WATCHER : \t   packet-size : %v\n", host.Size)
	}
	return nil
}

func run(log *log.Logger) error {
	log.Printf("WATCHER : started : pid %d", pid)
	defer log.Println("WATCHER: completed")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	appErrors := make(chan error, 1)

	conf, err := alive.ReadConfig(*configFile)
	if err != nil {
		return err
	}

	err = initWatchers(conf, log)
	if err != nil {
		return err
	}
	for _, watcher := range watchers {
		go func(w *alive.Watcher, appErrors chan error) {
			if err := w.Run(); err != nil {
				appErrors <- err
			}
		}(watcher, appErrors)
	}

	select {
	case err := <-appErrors:
		return err
	case sig := <-shutdown:
		log.Printf("WATCHER: %v : start shutdown", sig)
		for _, w := range watchers {
			log.Printf("WATCHER : stop : %s\n", w.Addr())
			w.Stop()
		}
	}

	return nil
}
