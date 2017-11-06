package main

import (
	"net/http"
	"encoding/json"
)

func Publish(addr string, exports []*Export) error {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		data := make(map[string]map[string]interface{}, len(exports))
		for _, export := range exports {
			data[export.Name] = map[string]interface{}{
				"topic":          export.Topic,
				"type":           export.Type,
				"database":       export.Database,
				"metric":         export.Metric,
				"tags":           export.Tags,
				"field":          export.Field,
				"last_value":     export.LastValue,
				"received_time":  export.ReceivedTime,
				"published_time": export.PublishedTime,
			}
		}

		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	server := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handler),
	}

	return server.ListenAndServe()
}
