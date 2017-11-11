package main

import (
	"time"
	"strconv"
	"log"
	"fmt"
	"strings"
	"text/template"
	"bytes"
	"github.com/eclipse/paho.mqtt.golang"
)

type Parser func(s string) (interface{}, error)

type Point struct {
	Metric string
	Tags   map[string]string
	Field  string
	Value  interface{}
	Time   time.Time
}

type Export struct {
	Name string

	Topic string

	Parser Parser

	Metric *template.Template
	Tags   map[string]*template.Template
	Field  *template.Template

	LastPoint Point

	ReceivedTime  *time.Time
	PublishedTime *time.Time

	interval time.Duration
	ticker   *time.Ticker

	o chan<- Point
}

func findParser(p string) (Parser, error) {
	switch p {
	case "string":
		return func(s string) (interface{}, error) {
			return s, nil
		}, nil

	case "bool":
		return func(s string) (interface{}, error) {
			// TODO: Specify values in config
			return strconv.ParseBool(s)
		}, nil

	case "int":
		return func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 0, 64)
		}, nil

	case "float":
		return func(s string) (interface{}, error) {
			return strconv.ParseFloat(s, 64)
		}, nil

	default:
		return nil, fmt.Errorf("parser not supported: %s", p)
	}
}

func explodePattern(s string) []string {
	is := strings.IndexRune(s, '{')
	ie := strings.IndexRune(s, '}')
	if is == -1 || ie == -1 || ie < is {
		return []string{s}
	}

	prefix := s[0:is]
	suffix := s[ie+1:]

	parts := strings.Split(s[is+1:ie], ",")

	var results []string
	for _, part := range parts {
		for _, result := range explodePattern(prefix + part + suffix) {
			results = append(results, result)
		}
	}

	return results
}

func BuildExports(cfg *Config, o chan<- Point) ([]*Export, error) {
	var exports []*Export

	for name := range cfg.Exports {
		parser, err := findParser(cfg.Exports[name].Parser)
		if err != nil {
			return nil, err
		}

		for _, topic := range explodePattern(cfg.Exports[name].Topic) {

			metric, err := template.New(name + ".metric").Parse(cfg.Exports[name].Metric)
			if err != nil {
				return nil, fmt.Errorf("invalid template: %v", err)
			}

			tags := make(map[string]*template.Template)
			for k, v := range cfg.Exports[name].Tags {
				tags[k], err = template.New(name + ".tag." + k).Parse(v)
				if err != nil {
					return nil, fmt.Errorf("invalid template: %v", err)
				}
			}

			field, err := template.New(name + ".field").Parse(cfg.Exports[name].Field)
			if err != nil {
				return nil, fmt.Errorf("invalid template: %v", err)
			}

			e := &Export{
				Name: name,

				Topic: topic,

				Parser: parser,

				Metric: metric,
				Tags:   tags,
				Field:  field,

				interval: cfg.Exports[name].Interval,
				ticker:   nil,

				o: o,
			}

			exports = append(exports, e)
		}
	}

	return exports, nil
}

func (e *Export) Stop() {
	if e.ticker != nil {
		e.ticker.Stop()
	}
}

func interpolate(t *template.Template, ctx map[string]interface{}) (string, error) {
	var out bytes.Buffer
	if err := t.Execute(&out, ctx); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (e *Export) Handle(c mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message on %s: %s", msg.Topic(), msg.Payload())

	now := time.Now()

	value, err := e.Parser(string(msg.Payload()))
	if err != nil {
		log.Printf("Failed to parse message: %s: %v", msg, err)
	}

	context := map[string]interface{}{
		"topic": strings.Split(msg.Topic(), "/"),
		"value": value,
	}

	metric, err := interpolate(e.Metric, context)
	if err != nil {
		log.Print(err)
		return
	}

	tags := make(map[string]string)
	for k, v := range e.Tags {
		tags[k], err = interpolate(v, context)
		if err != nil {
			log.Print(err)
			return
		}
	}

	field, err := interpolate(e.Field, context)
	if err != nil {
		log.Print(err)
		return
	}

	point := Point{
		Metric: metric,
		Tags:   tags,
		Field:  field,
		Value:  value,
		Time:   now,
	}

	e.LastPoint = point
	e.ReceivedTime = &now

	e.o <- point

	if e.ticker != nil {
		e.ticker.Stop()
	}
	if e.interval != 0 {
		e.ticker = time.NewTicker(e.interval)
		go func() {
			for range e.ticker.C {
				point.Time = time.Now()
				e.o <- point
			}
		}()
	}
}
