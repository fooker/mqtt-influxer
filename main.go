package main

import (
	"fmt"
	"log"
	"github.com/eclipse/paho.mqtt.golang"
	influxdb "github.com/influxdata/influxdb/client/v2"
	"time"
	"flag"
)

var config_flag = flag.String("config", "config.yaml", "Path to the config file")
var http_flag = flag.String("http-port", "", "Port for http status export (leave empty to disable)")

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

	// Create exports
	exports := make([]*Export, 0)
	for name, config := range config.Exports {
		export, err := NewExport(name, config)
		if err != nil {
			log.Fatalf("Failed to create export: %v", err)
		}

		defer export.Stop()

		if t := mqttClient.Subscribe(config.Topic, 0, func(client mqtt.Client, message mqtt.Message) {
			log.Printf("Received message on %s: %s", message.Topic(), message.Payload())

			export.I <- string(message.Payload())

		}); t.Wait() && t.Error() != nil {
			log.Fatalf("Failed to subscribe %s: %v", config.Topic, t.Error())
		} else {
			log.Printf("Subscribed to %s", config.Topic)
		}

		go func() {
			for value := range export.O {
				points, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
					Database:  export.Database,
					Precision: "us",
				})

				point, err := influxdb.NewPoint(
					export.Metric,
					export.Tags,
					map[string]interface{}{export.Field: value},
					time.Now())
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

		exports = append(exports, export)
	}

	if *http_flag != "" {
		Publish(*http_flag, exports)
		
	} else {
		select {}
	}
}
