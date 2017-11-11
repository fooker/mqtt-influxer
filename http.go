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
			//tags := make(map[string]string)
			//for k, v := range export.Tags {
			//	tags[k] = v.DefinedTemplates()
			//}

			data[export.Name] = map[string]interface{}{
				"topic":  export.Topic,
				//"parser": export.Parser,
				//"metric": export.Metric.Name(),
				//"tags":   tags,
				//"field":  export.Field.DefinedTemplates(),
				"last_point": map[string]interface{}{
					"metric": export.LastPoint.Metric,
					"tags":   export.LastPoint.Tags,
					"field":  export.LastPoint.Field,
					"value":  export.LastPoint.Value,
				},
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
