package main

import (
	"time"
	"strconv"
	"log"
	"fmt"
)

type Export struct {
	Name string

	Topic string
	Type  string

	Database string
	Metric   string
	Tags     map[string]string
	Field    string

	LastValue interface{}

	ReceivedTime  *time.Time
	PublishedTime *time.Time

	interval time.Duration
	ticker   *time.Ticker

	I chan<- string
	O <-chan interface{}
}

func NewExport(name string, cfg *ExportConfig) (*Export, error) {
	var parser func(s string) (interface{}, error)

	switch cfg.Type {
	case "string":
		parser = func(s string) (interface{}, error) {
			return s, nil
		}

	case "bool":
		parser = func(s string) (interface{}, error) {
			// TODO: Specify values in config
			return strconv.ParseBool(s)
		}

	case "int":
		parser = func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 0, 64)
		}

	case "float":
		parser = func(s string) (interface{}, error) {
			return strconv.ParseFloat(s, 64)
		}

	default:
		return nil, fmt.Errorf("type not supported: %s", cfg.Type)
	}

	i := make(chan string)
	o := make(chan interface{})

	e := &Export{
		Name: name,

		Topic: cfg.Topic,
		Type:  cfg.Type,

		Database: cfg.Database,
		Metric:   cfg.Metric,
		Tags:     cfg.Tags,
		Field:    cfg.Field,

		interval: cfg.Interval,
		ticker:   nil,

		I: i,
		O: o,
	}

	go e.handle(i, o, parser)

	return e, nil
}

func (e *Export) Stop() {
	if e.ticker != nil {
		e.ticker.Stop()
	}
}

func (e *Export) handle(i <-chan string, o chan<- interface{}, parser func(s string) (interface{}, error)) {
	for s := range i {
		now := time.Now()

		value, err := parser(s)
		if err != nil {
			log.Printf("Failed to parse value: %s: %v", s, err)
			continue
		}

		e.LastValue = value
		e.ReceivedTime = &now

		o <- value

		if e.ticker != nil {
			e.ticker.Stop()
		}
		if e.interval != 0 {
			e.ticker = time.NewTicker(e.interval)
			go func() {
				for range e.ticker.C {
					o <- value
				}
			}()
		}
	}
}
