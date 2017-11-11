package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

type MQTTConfig struct {
	Address  string `yaml:"address"`
	ClientID string `yaml:"client_id"`
}

type InfluxConfig struct {
	Address  string `yaml:"address"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ExportConfig struct {
	Topic string `yaml:"topic"`

	Parser string `yaml:"parser"`
	Script string `yaml:"script"`

	Metric string            `yaml:"metric"`
	Tags   map[string]string `yaml:"tags"`
	Field  string            `yaml:"field"`

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
			if config.Exports[name].Parser == "" {
				config.Exports[name].Parser = "string"
			}

			if config.Exports[name].Metric == "" {
				config.Exports[name].Metric = name
			}

			if config.Exports[name].Tags == nil {
				config.Exports[name].Tags = make(map[string]string)
			}

			if config.Exports[name].Field == "" {
				config.Exports[name].Field = "value"
			}
	}

	log.Printf("Config: Loaded: %v", config)

	return config, nil
}
