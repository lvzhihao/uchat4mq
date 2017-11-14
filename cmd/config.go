package cmd

import (
	"errors"
	"os"

	"go.uber.org/zap"

	"github.com/lvzhihao/uchat4mq/rmqtool"
	"github.com/spf13/viper"
)

type Config struct {
	Consumer struct {
		Conn  rmqtool.ConnectConfig
		Queue rmqtool.QueueConfig
	}
	Publisher struct {
		Conn     rmqtool.ConnectConfig
		Exchange string
		Key      string
	}
}

func LoadConfig(key string) (*Config, error) {
	config := &Config{}
	err := config.Load(key)
	return config, err
}

func GetLogger() *zap.Logger {
	// zap logger
	var logger *zap.Logger
	if os.Getenv("DEBUG") == "true" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	return logger
}

func (c *Config) Load(key string) error {
	return viper.UnmarshalKey(key, c)
}

func (c *Config) ConsumerQueue() (*rmqtool.Queue, error) {
	if c.ConsumerQueueName() == "" {
		return nil, errors.New("empty consumer queue name")
	} else {
		conn := rmqtool.Conn(c.Consumer.Conn)
		queue := conn.ApplyQueue(c.ConsumerQueueName())
		err := queue.Ensure(c.Consumer.Queue.Bindlist)
		return queue, err
	}
}

func (c *Config) ConsumerQueueName() string {
	return c.Consumer.Queue.Name
}

func (c *Config) PublisherTool() (*rmqtool.PublisherTool, error) {
	conn := rmqtool.Conn(c.Publisher.Conn)
	if c.Publisher.Key == "" {
		return nil, errors.New("empty publisher key")
	} else if err := conn.CreateExchange(c.PublisherExchange()); err != nil {
		return nil, err
	} else {
		return conn.ApplyPublisher(c.PublisherExchange(), []string{c.PublisherKey()})
	}
}

func (c *Config) PublisherExchange() string {
	return c.Publisher.Exchange
}

func (c *Config) PublisherKey() string {
	return c.Publisher.Key
}
