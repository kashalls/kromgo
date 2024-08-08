package kromgo

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type EndpointResponse struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color,omitempty"`
	Error         bool   `json:"isError,omitempty"`
	Style         string `json:"style,omitempty"`
}

func HandleError(w http.ResponseWriter, r *http.Request, metric string, reason string) {
	response := EndpointResponse{
		SchemaVersion: 1,
		Label:         metric,
		Message:       reason,
		Error:         true,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error converting data to json response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
