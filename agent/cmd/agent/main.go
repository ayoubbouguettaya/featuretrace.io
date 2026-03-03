package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"featuretrace.io/agent/internals/config"
	"featuretrace.io/agent/internals/pipeline"
)

func main() {
	cfgPath := flag.String("config", "", "path to config.yaml")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("[agent] FeatureTrace Agent starting…")

	// --- Load configuration ---
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("[agent] failed to load config: %v", err)
	}

	log.Printf("[agent] config: flush=%s batch=%d endpoint=%s",
		cfg.Agent.FlushInterval, cfg.Agent.BatchSize, cfg.Output.Endpoint)

	// --- Build and start the pipeline ---
	p := pipeline.New(cfg)
	p.Start()

	// --- Block until OS signal (SIGINT / SIGTERM) ---
	waitForShutdown()

	// --- Graceful drain ---
	log.Println("[agent] received shutdown signal")
	p.Stop()
	log.Println("[agent] exited cleanly")
}

// waitForShutdown blocks until the process receives SIGINT or SIGTERM.
func waitForShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
