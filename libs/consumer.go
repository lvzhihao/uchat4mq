package libs

import (
	"time"

	"github.com/lvzhihao/goutils"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type receiveConsumer struct {
	amqpUrl string
	conn    *amqp.Connection
	logger  *zap.Logger
}

func NewReceiveConsumer(url string, logger *zap.Logger) (*receiveConsumer, error) {
	c := &receiveConsumer{
		amqpUrl: url,
		logger:  logger,
	}
	// first test dial
	_, err := amqp.Dial(url)
	return c, err
}

func (c *receiveConsumer) link(queue string, prefetchCount int) (<-chan amqp.Delivery, error) {
	var err error
	c.conn, err = amqp.Dial(c.amqpUrl)
	if err != nil {
		c.logger.Error("amqp.open", zap.Error(err))
		return nil, err
	}
	_, err = c.conn.Channel()
	if err != nil {
		c.logger.Error("channel.open", zap.Error(err))
		return nil, err
	}
	channel, _ := c.conn.Channel()
	if err := channel.Qos(prefetchCount, 0, false); err != nil {
		c.logger.Error("channel.qos", zap.Error(err))
		return nil, err
	}
	deliveries, err := channel.Consume(queue, "ctag-"+goutils.RandomString(20), false, false, false, false, nil)
	if err != nil {
		c.logger.Error("base.consume", zap.Error(err))
		return deliveries, err
	}
	return deliveries, nil
}

func (c *receiveConsumer) Consumer(queue string, prefetchCount int, handle func(msg amqp.Delivery, logger *zap.Logger)) {
	for {
		time.Sleep(3 * time.Second)
		deliveries, err := c.link(queue, prefetchCount)
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
		c.conn.Close()
		c.logger.Info("Consumer ReConnection After 3 Second")
	}
}
