package uchat4mq

import (
	"time"

	"github.com/lvzhihao/goutils"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type ConsumerTool struct {
	amqpUrl   string
	logger    *zap.Logger
	RetryTime time.Duration
}

func NewConsumerTool(url string, logger *zap.Logger) (*ConsumerTool, error) {
	c := &ConsumerTool{
		amqpUrl:   url,
		logger:    logger,
		RetryTime: time.Second * 3, //default retry
	}
	// first test dial
	_, err := amqp.Dial(url)
	return c, err
}

func (c *ConsumerTool) link(queue string, prefetchCount int) (*amqp.Connection, <-chan amqp.Delivery, error) {
	conn, err := amqp.Dial(c.amqpUrl)
	if err != nil {
		c.logger.Error("amqp.open", zap.Error(err))
		return nil, nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		c.logger.Error("channel.open", zap.Error(err))
		conn.Close()
		return nil, nil, err
	}
	if err := channel.Qos(prefetchCount, 0, false); err != nil {
		c.logger.Error("channel.qos", zap.Error(err))
		conn.Close()
		return nil, nil, err
	}
	deliveries, err := channel.Consume(queue, "ctag-"+goutils.RandomString(20), false, false, false, false, nil)
	if err != nil {
		c.logger.Error("base.consume", zap.Error(err))
		conn.Close()
		return nil, nil, err
	}
	return conn, deliveries, nil
}

func (c *ConsumerTool) Consume(queue string, prefetchCount int, handle func(msg amqp.Delivery, logger *zap.Logger)) {
	for {
		time.Sleep(c.RetryTime)
		conn, deliveries, err := c.link(queue, prefetchCount)
		if err != nil {
			c.logger.Error("Consumer Link Error", zap.Error(err))
			continue
		}
		for msg := range deliveries {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						c.logger.Error("process recover", zap.Any("panic", r))
					}
				}()
				handle(msg, c.logger)
			}()
		}
		conn.Close()
		c.logger.Info("Consumer ReConnection After RetryTime", zap.Duration("retryTime", c.RetryTime))
	}
}
