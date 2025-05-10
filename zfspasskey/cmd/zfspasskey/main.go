package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"filippo.io/mostly-harmless/zfspasskey"
	"gopkg.in/yaml.v3"
)

func main() {
	configFlag := flag.String("c", "", "path to the config file")
	listenFlag := flag.String("l", ":8080", "address to listen on")
	certFlag := flag.String("cert", "", "path to the TLS certificate")
	keyFlag := flag.String("key", "", "path to the TLS key")
	flag.Parse()

	configYAML, err := os.ReadFile(*configFlag)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	var datasets map[string]string
	if err := yaml.Unmarshal(configYAML, &datasets); err != nil {
		log.Fatalf("Failed to unmarshal config file: %v", err)
	}

	handler, err := zfspasskey.NewHandler(datasets)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}
	server := &http.Server{
		Addr:         *listenFlag,
		Handler:      handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	log.Printf("Starting server on %s", *listenFlag)
	log.Println(server.ListenAndServeTLS(*certFlag, *keyFlag))
}
