package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
)

type MetricColor struct {
	Min           float64 `yaml:"min" json:"min"`
	Max           float64 `yaml:"max" json:"max"`
	Color         string  `yaml:"color,omitempty" json:"color,omitempty"`
	ValueOverride string  `yaml:"valueOverride,omitempty" json:"valueOverride,omitempty"`
}

type Metric struct {
	Name   string        `yaml:"name" json:"name"`
	Query  string        `yaml:"query" json:"query"`
	Label  string        `yaml:"label,omitempty" json:"label,omitempty"`
	Prefix string        `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Suffix string        `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	Colors []MetricColor `yaml:"colors,omitempty" json:"colors,omitempty"`
}

type Config struct {
	Debug      bool     `yaml:"debug,omitempty" json:"debug,omitempty"`
	Prometheus string   `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Metrics    []Metric `yaml:"metrics" json:"metrics"`
}

type MetricResult struct {
	Metric map[string]interface{} `json:"metric"`
	Value  []interface{}          `json:"value"`
}

var configPath = "/kromgo/config.yaml" // Default config file path

func main() {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Check if a custom config file path is provided via command line argument
	configPathFlag := flag.String("config", "", "Path to the YAML config file")
	jsonSchemaFlag := flag.Bool("jsonschema", false, "Dump JSON Schema for config file")
	flag.Parse()

	if *jsonSchemaFlag {
		jsonString, _ := json.MarshalIndent(jsonschema.Reflect(&Config{}), "", "  ")
		fmt.Println(string(jsonString))
		return
	}

	if *configPathFlag != "" {
		configPath = *configPathFlag
	}

	// Load the YAML config file
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %s\n", err)
		os.Exit(1)
	}

	if config.Debug {
		logLevel.Set(slog.LevelDebug)
	}

	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL != "" {
		config.Prometheus = prometheusURL
	}

	if len(config.Prometheus) == 0 {
		panic("No valid prometheus endpoint was set in config or environment.")
	}

	// Create a Prometheus API client
	client, err := api.NewClient(api.Config{
		Address: config.Prometheus,
	})
	if err != nil {
		fmt.Printf("Error creating Prometheus client: %s\n", err)
		os.Exit(1)
	}

	// Create a Prometheus v1 API client
	v1api := v1.NewAPI(client)

	// Set up HTTP server
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {

		slog.Info("incoming request",
			slog.String("method", r.Method),
			slog.String("ip", r.RemoteAddr),
			slog.String("url", r.URL.String()),
		)

		// Get the metric name from the query parameter
		metricName := r.URL.Query().Get("metric")
		responseFormat := r.URL.Query().Get("format")

		// Find the corresponding metric configuration
		var metric Metric
		for _, configMetric := range config.Metrics {
			if configMetric.Name == metricName {
				metric = configMetric
				break
			}
		}

		// If metric not found, return an error
		if metric.Query == "" {
			slog.Error(
				"metric not found",
				slog.String("ip", r.RemoteAddr),
				slog.String("metric", metric.Name),
			)
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}

		// Run the Prometheus query
		result, warnings, err := v1api.Query(r.Context(), metric.Query, time.Now())
		if err != nil {
			slog.Error(
				"error executing query",
				slog.String("ip", r.RemoteAddr),
				slog.String("metric", metric.Name),
				"error", err,
			)
			http.Error(w, fmt.Sprintf("Error executing query: %s", err), http.StatusInternalServerError)
			return
		}

		if len(warnings) > 0 {
			fmt.Println("Warnings while executing query:", warnings)
		}

		// Convert the result to JSON
		jsonResult, err := json.Marshal(result)
		slog.Debug(
			"query result",
			slog.String("ip", r.RemoteAddr),
			slog.String("metric", metric.Name),
			slog.String("query", metric.Query),
			slog.String("result", string(jsonResult)),
		)
		if err != nil {
			slog.Error(
				"could not convert to json",
				slog.String("ip", r.RemoteAddr),
				slog.String("metric", metric.Name),
				"error", err,
			)
			http.Error(w, fmt.Sprintf("Error converting result to JSON: %s", err), http.StatusInternalServerError)
			return
		}

		if len(jsonResult) <= 0 {
			slog.Error(
				"query returned no results",
				slog.String("ip", r.RemoteAddr),
				slog.String("metric", metric.Name),
				slog.String("query", metric.Query),
			)
			http.Error(w, "Query returned no results", http.StatusNotFound)
			return
		}

		if (responseFormat == "raw") {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResult)
			return
		} else {

			responseResult := result.(model.Vector)
			resultValue := float64(responseResult[0].Value)
			colorConfig := getColorConfig(metric.Colors, resultValue)

			var whatAmIShowing string = strconv.FormatFloat(resultValue, 'f', -1, 64)

			if len(metric.Label) > 0 {
				value, err := ExtractLabelValue(responseResult, metric.Label)
				if err != nil {
					http.Error(w, "Label was not present in query.", http.StatusBadGateway)
					slog.Error(
						"label was not found in query result",
						slog.String("ip", r.RemoteAddr),
						slog.String("metric", metric.Name),
						"label", metric.Label,
					)
					return
				}
				whatAmIShowing = value
			}

			if len(colorConfig.ValueOverride) > 0 {
				whatAmIShowing = colorConfig.ValueOverride
			}

			message := metric.Prefix + whatAmIShowing + metric.Suffix

			data := map[string]interface{}{
				"schemaVersion": 1,
				"label":         metricName,
				"message":       message,
			}

			if colorConfig.Color != "" {
				data["color"] = colorConfig.Color
			}

			// Convert the data to JSON
			jsonData, err := json.Marshal(data)
			if err != nil {
				http.Error(w, "Error converting to JSON", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonData)

		}
	})

	// Determine the HTTP server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	// Start the HTTP server
	slog.Info("server is listening",
		slog.String("port", port),
	)
	http.ListenAndServe(":"+port, nil)
}

// Load the YAML config file
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %s", err)
	}

	return &config, nil
}

func getColorConfig(colors []MetricColor, value float64) MetricColor {
	for _, colorConfig := range colors {
		if value >= colorConfig.Min && value <= colorConfig.Max {
			return colorConfig
		}
	}

	// MetricColors is enabled, but the value does not have a corresponding value to it.
	// We return a default value here only if the result value falls outside the range.
	return MetricColor{
			Min: value,
			Max: value,
	}
}

func ExtractLabelValue(vector model.Vector, labelName string) (string, error) {
	// Extract label value from the first sample of the result
	if len(vector) > 0 {
		// Check if the label exists in the first sample
		if val, ok := vector[0].Metric[model.LabelName(labelName)]; ok {
			return string(val), nil
		}
	}

	// If label not found, return an error
	return "", fmt.Errorf("label '%s' not found in the query result", labelName)
}
