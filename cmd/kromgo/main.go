package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/log"
	"github.com/kashalls/kromgo/cmd/kromgo/init/prometheus"
	"github.com/kashalls/kromgo/cmd/kromgo/init/server"

	"go.uber.org/zap"
)

const banner = `
kromgo
version: %s (%s)

`

var (
	Version = "local"
	Gitsha  = "?"
)

func main() {
	fmt.Printf(banner, Version, Gitsha)

	jsonSchemaFlag := flag.Bool("jsonschema", false, "Dump JSON Schema for config file")
	flag.Parse()

	if *jsonSchemaFlag {
		jsonString, _ := json.MarshalIndent(jsonschema.Reflect(&configuration.Config{}), "", "  ")
		fmt.Println(string(jsonString))
		return
	}

	log.Init()

	config := configuration.Init()
	_, err := prometheus.Init(config)
	if err != nil {
		log.Error("failed to initialize provider", zap.Error(err))
	}

	main, health := server.Init(config)
	server.ShutdownGracefully(main, health)
}
