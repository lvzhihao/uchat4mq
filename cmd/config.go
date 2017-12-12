package cmd

import (
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	rmqtool "github.com/lvzhihao/go-rmqtool"
	"github.com/spf13/viper"
)

type ZapLoggerWrapperForRmqtool struct {
	rmqtool.LoggerInterface
	Logger *zap.Logger
}

func (c *ZapLoggerWrapperForRmqtool) Field(input ...interface{}) []zapcore.Field {
	ret := make([]zapcore.Field, 0)
	for k, v := range input {
		switch v.(type) {
		case zapcore.Field:
			ret = append(ret, v.(zapcore.Field))
		default:
			ret = append(ret, zap.Any(fmt.Sprintf("Field_%d", k+1), v))
		}
	}
	return ret
}

func (c *ZapLoggerWrapperForRmqtool) Error(format string, v ...interface{}) {
	c.Logger.Error(format, c.Field(v...)...)
}

func (c *ZapLoggerWrapperForRmqtool) Debug(format string, v ...interface{}) {
	c.Logger.Debug(format, c.Field(v...)...)
}

func (c *ZapLoggerWrapperForRmqtool) Warn(format string, v ...interface{}) {
	c.Logger.Warn(format, c.Field(v...)...)
}

func (c *ZapLoggerWrapperForRmqtool) Info(format string, v ...interface{}) {
	c.Logger.Info(format, c.Field(v...)...)
}

func (c *ZapLoggerWrapperForRmqtool) Fatal(format string, v ...interface{}) {
	c.Logger.Fatal(format, c.Field(v...)...)
}

func (c *ZapLoggerWrapperForRmqtool) Panic(format string, v ...interface{}) {
	c.Logger.Panic(format, c.Field(v...)...)
}

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

func GetZapLoggerWrapperForRmqtool(logger *zap.Logger) *ZapLoggerWrapperForRmqtool {
	return &ZapLoggerWrapperForRmqtool{
		Logger: logger,
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
		conn := rmqtool.NewConnect(c.Consumer.Conn)
		queue := conn.ApplyQueue(c.ConsumerQueueName())
		err := queue.Ensure(c.Consumer.Queue.Bindlist)
		return queue, err
	}
}

func (c *Config) ConsumerQueueName() string {
	return c.Consumer.Queue.Name
}

func (c *Config) PublisherTool() (*rmqtool.PublisherTool, error) {
	conn := rmqtool.NewConnect(c.Publisher.Conn)
	if c.Publisher.Key == "" {
		return nil, errors.New("empty publisher key")
	} else if err := conn.QuickCreateExchange(c.PublisherExchange(), "topic", true); err != nil {
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
