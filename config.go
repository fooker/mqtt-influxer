package main

import (
	"gopkg.in/yaml.v2"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

type MQTTConfig struct {
	Address  string `yaml:"address"`
	ClientID string `yaml:"client_id"`
	Realm    string `yaml:"realm"`
}

type InfluxConfig struct {
	Address  string `yaml:"address"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ExportConfig struct {
	Topic string `yaml:"topic"`
	Type  string `yaml:"type"`

	Database string            `yaml:"database"`
	Metric   string            `yaml:"metric"`
	Tags     map[string]string `yaml:"tags"`
	Field    string            `yaml:"field"`

	Interval time.Duration `yaml:"interval"`

	//Options map[string]interface{} `yaml:",inline"`
}

type Config struct {
	MQTT     *MQTTConfig   `yaml:"mqtt"`
	InfluxDB *InfluxConfig `yaml:"influxdb"`

	Exports map[string]*ExportConfig `yaml:"exports"`
}

func LoadConfig(filename string) (*Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(b, config); err != nil {
		return nil, err
	}

	for name := range config.Exports {
		if config.MQTT.Realm != "" {
			config.Exports[name].Topic = fmt.Sprintf("%s/%s", config.MQTT.Realm, config.Exports[name].Topic)
		}

		if config.Exports[name].Type == "" {
			config.Exports[name].Type = "float"
		}

		if config.Exports[name].Database == "" {
			config.Exports[name].Database = config.InfluxDB.Database
		}

		if config.Exports[name].Metric == "" {
			config.Exports[name].Metric = name
		}

		if config.Exports[name].Tags == nil {
			config.Exports[name].Tags = make(map[string]string)
		}

		if _, found := config.Exports[name].Tags["name"]; !found {
			config.Exports[name].Tags["name"] = name
		}

		if _, found := config.Exports[name].Tags["topic"]; !found {
			config.Exports[name].Tags["topic"] = config.Exports[name].Topic
		}

		if config.Exports[name].Field == "" {
			config.Exports[name].Field = "value"
		}
	}

	log.Printf("Config: Loaded: %v", config)

	return config, nil
}
