package uchat4mq

import (
	"time"

	"github.com/lvzhihao/goutils"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type PublisherTool struct {
	logger    *zap.Logger
	channels  map[string]*publishChannel
	RetryTime time.Duration
}

func NewPublisherTool(url, exchange string, routeKeys []string, logger *zap.Logger) (*PublisherTool, error) {
	tool := &PublisherTool{
		logger:    logger,
		channels:  make(map[string]*publishChannel, 0),
		RetryTime: time.Second * 3, //default retry
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
			Channel:  make(chan interface{}, 1000),
		}
		go c.channels[route].Receive()
	}
	return nil
}

func (c *PublisherTool) publish(route string, msg interface{}) {
	if s, ok := c.channels[route]; ok {
		s.Channel <- msg
	}
}

func (c *PublisherTool) Publish(route string, msg amqp.Publishing) {
	c.publish(route, msg)
}

func (c *PublisherTool) PublishExt(route, fix string, msg amqp.Publishing) {
	c.publish(route, &publishingExt{
		routeKeyFix: fix,
		msg:         msg,
	})
}

type publishingExt struct {
	routeKeyFix string
	msg         amqp.Publishing
}

func (c *publishingExt) Key(prefix string) string {
	return prefix + c.routeKeyFix
}

func (c *publishingExt) Msg() amqp.Publishing {
	return c.msg
}

type publishChannel struct {
	logger    *zap.Logger
	amqpUrl   string
	exchange  string
	routeKey  string
	retryTime time.Duration
	Channel   chan interface{}
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
			switch msg.(type) {
			case string:
				if goutils.ToString(msg) == "quit" {
					c.logger.Info("Channel Connection Quit", zap.String("route", c.routeKey))
					conn.Close()
					return
				} //quit
			case amqp.Publishing:
				err := channel.Publish(c.exchange, c.routeKey, false, false, msg.(amqp.Publishing))
				if err != nil {
					c.Channel <- msg
					conn.Close()
					c.logger.Error("Channel Connection Error 4", zap.String("route", c.routeKey), zap.Error(err))
					break BreakFor
				}
			case *publishingExt:
				err := channel.Publish(c.exchange, msg.(*publishingExt).Key(c.routeKey), false, false, msg.(*publishingExt).Msg())
				if err != nil {
					c.Channel <- msg
					conn.Close()
					c.logger.Error("Channel Connection Error 4", zap.String("route", c.routeKey), zap.Error(err))
					break BreakFor
				}
			}
		}
	}
	time.Sleep(3 * time.Second)
	goto RetryConnect
}
