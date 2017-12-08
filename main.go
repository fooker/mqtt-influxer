package main

import (
	"fmt"
	"log"
	"github.com/eclipse/paho.mqtt.golang"
	influxdb "github.com/influxdata/influxdb/client/v2"
	"flag"
	"os"
	"os/signal"
)

var configFlag = flag.String("config", "config.yaml", "Path to the config file")

func main() {
	flag.Parse()

	// Load config
	config, err := LoadConfig(*configFlag)
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

	// Spawn export handlers
	o := make(chan Point, 10)
	go func() {
		for point := range o {
			if point.Values == nil || len(point.Values) == 0 {
				log.Printf("Skipping empty point: %v", point)
				continue
			}

			points, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
				Database:  config.InfluxDB.Database,
				Precision: "us",
			})

			point, err := influxdb.NewPoint(
				point.Metric,
				point.Tags,
				point.Values,
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

	// Wait for termination
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	// Stop all exporters
	for _, export := range exports {
		export.Stop()
	}
}
