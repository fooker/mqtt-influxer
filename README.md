# mqtt-influxer
The `mqtt-influxer` exports metrices and other values received from a `MQTT` broker to an `InfluxDB` database.
It is capable to subscribe to multiple `MQTT` topics and parse the received message according to a given value type.
The received values are then inserted into a `InfluxDB` instance whereas each subscribed topic is associated with the according database and metric settings.

## Installation
The software is writen in go and therefore requires a functional go environment.

```
go get git.maglab.space/fooker/mqtt-influxer
```

## Configuration
The configuration is written in a single `.yaml` file.
The file consists of three main keys:
`mqtt` - the configuration used to connect to the `MQTT` broker;
`influxdb` - the configuration used to connect to the `InfluxDB` database;
`exports` - the configuration for the subscribed topics and the according value processing and insertion.

See `config.yaml.example` for a detailed example.

## Usage
After installation and creation of the config file, the application can be started using the following command:
```
./mqtt-influxer -config config.yaml
```

## Republishing interval
For values which do not change frequently, is can be useful to reinsert the last value on a regular interval.
By specifying the `interval` parameter in en export configuration, the daemon will reinsert the last received value on the given interval.
