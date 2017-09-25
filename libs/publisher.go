package libs

import (
	"time"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type PublisherTool struct {
	logger   *zap.Logger
	channels map[string]*publishChannel
}

func NewPublisherTool(logger *zap.Logger, url, exchange string, routeKeys []string) (*PublisherTool, error) {
	tool := &PublisherTool{
		logger:   logger,
		channels: make(map[string]*publishChannel, 0),
	}
	err := tool.conn(url, exchange, routeKeys)
	return tool, err
}

func (c *PublisherTool) conn(url, exchange string, routeKeys []string) error {
	_, err := amqp.Dial(url)
	if err != nil {
		return err
	} //test link
	for _, route := range routeKeys {
		c.channels[route] = &publishChannel{
			logger:   c.logger,
			amqpUrl:  url,
			exchange: exchange,
			routeKey: route,
			Channel:  make(chan amqp.Publishing, 1000),
		}
		go c.channels[route].Receive()
	}
	return nil
}

func (c *PublisherTool) Publish(route string, msg amqp.Publishing) {
	c.channels[route].Channel <- msg
}

type publishChannel struct {
	logger   *zap.Logger
	amqpUrl  string
	exchange string
	routeKey string
	Channel  chan amqp.Publishing
}

func (c *publishChannel) Receive() {
RetryConnect:
	conn, err := amqp.Dial(c.amqpUrl)
	if err != nil {
		c.logger.Error("Channel Connection Error 1", zap.String("route", c.routeKey), zap.Error(err))
		time.Sleep(3 * time.Second)
		goto RetryConnect
	}
	channel, err := conn.Channel()
	if err != nil {
		c.logger.Error("Channel Connection Error 2", zap.String("route", c.routeKey), zap.Error(err))
		conn.Close()
		time.Sleep(3 * time.Second)
		goto RetryConnect
	}
	err = channel.ExchangeDeclare(c.exchange, "topic", true, false, false, false, nil)
	if err != nil {
		c.logger.Error("Channel Connection Error 3", zap.String("route", c.routeKey), zap.Error(err))
		conn.Close()
		time.Sleep(3 * time.Second)
		goto RetryConnect
	}
BreakFor:
	for {
		select {
		case msg := <-c.Channel:
			if string(msg.Body) == "quit" {
				c.logger.Info("Channel Connection Quit", zap.String("route", c.routeKey))
				conn.Close()
				return
			} //quit
			err := channel.Publish(c.exchange, c.routeKey, false, false, msg)
			if err != nil {
				c.Channel <- msg
				conn.Close()
				c.logger.Error("Channel Connection Error 4", zap.String("route", c.routeKey), zap.Error(err))
				break BreakFor
			}
		}
	}
	goto RetryConnect
}
