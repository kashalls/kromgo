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
	configPathFlag := flag.String("config", "", "Path to the YAML config file")
	jsonSchemaFlag := flag.Bool("jsonschema", false, "Dump JSON Schema for config file")
	flag.Parse()

	if *jsonSchemaFlag {
		jsonString, _ := json.MarshalIndent(jsonschema.Reflect(&configuration.KromgoConfig{}), "", "  ")
		fmt.Println(string(jsonString))
		return
	}

	fmt.Printf(banner, Version, Gitsha)
	log.Init()

	config := configuration.Init(*configPathFlag)
	serverConfig := configuration.InitServer()
	_, err := prometheus.Init(config)
	if err != nil {
		log.Error("failed to initialize prometheus", zap.Error(err))
	}

	main, health := server.Init(config, serverConfig)
	server.ShutdownGracefully(main, health)
}
