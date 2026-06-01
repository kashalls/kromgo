// Command schema prints the JSON Schema for kromgo's config file to stdout.
// It is a dev/CI tool kept separate from the runtime binary so the JSON Schema
// reflection dependency is not linked into kromgo itself.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/invopop/jsonschema"
)

func main() {
	schema := jsonschema.Reflect(&config.KromgoConfig{})
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error generating schema:", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
