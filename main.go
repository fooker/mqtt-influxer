package main

import (
	"fmt"
	"log"
	"github.com/eclipse/paho.mqtt.golang"
	influxdb "github.com/influxdata/influxdb/client/v2"
	"flag"
)

var config_flag = flag.String("config", "config.yaml", "Path to the config file")
var http_flag = flag.String("http", "", "Listening address for http status export (leave empty to disable)")

func main() {
	flag.Parse()

	config, err := LoadConfig(*config_flag)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Connect to MQTT
	mqttClient := mqtt.NewClient(mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s/", config.MQTT.Address)).
		SetClientID(config.MQTT.ClientID))

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	} else {
		log.Printf("Connected to MQTT broker: %s", config.MQTT.Address)
	}

	defer mqttClient.Disconnect(0)

	// Connect to InfluxDB
	influxdbClient, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s/", config.InfluxDB.Address),
		Username: config.InfluxDB.Username,
		Password: config.InfluxDB.Password})
	if err != nil {
		log.Fatalf("Failed to connect to InfluxDB: %v", err)
	} else {
		log.Printf("Connected to InfluxDB: %s", config.InfluxDB.Address)
	}

	defer influxdbClient.Close()

	o := make(chan Point, 10)
	go func() {
		for point := range o {
			log.Printf("Storing point: %v", point)

			points, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
				Database:  config.InfluxDB.Database,
				Precision: "us",
			})

			point, err := influxdb.NewPoint(
				point.Metric,
				point.Tags,
				map[string]interface{}{point.Field: point.Value},
				point.Time,
			)
			if err != nil {
				log.Printf("Invalid data point: %v", err)
			}

			points.AddPoint(point)

			err = influxdbClient.Write(points)
			if err != nil {
				log.Printf("Failed to insert data point: %v", err)
			}
		}
	}()

	// Create exports
	exports, err := BuildExports(config, o)
	if err != nil {
		log.Fatalf("Failed to build exports: %v", err)
	}

	// Subscribe to exports
	for _, export := range exports {
		if t := mqttClient.Subscribe(export.Topic, 0, export.Handle); t.Wait() && t.Error() != nil {
			log.Fatalf("Failed to subscribe %s: %v", export.Topic, t.Error())
		} else {
			log.Printf("Subscribed to %s", export.Topic)
		}
	}

	if *http_flag != "" {
		Publish(*http_flag, exports)

	} else {
		select {}
	}

	for _, export := range exports {
		export.Stop()
	}
}
