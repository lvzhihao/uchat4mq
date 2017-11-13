// Copyright © 2017 edwin <edwin.lzh@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/lvzhihao/uchat4mq/rmqtool"
	"github.com/lvzhihao/uchatlib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
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

func (c *Config) Load(key string) error {
	return viper.UnmarshalKey(key, c)
}

func (c *Config) ConsumerQueue() (*rmqtool.Queue, error) {
	if c.Consumer.Queue.Name == "" {
		return nil, errors.New("empty consumer queue name")
	} else {
		conn := rmqtool.Conn(c.Consumer.Conn)
		queue := conn.ApplyQueue(c.Consumer.Queue.Name)
		err := queue.Ensure(c.Consumer.Queue.Bindlist)
		return queue, err
	}
}

func (c *Config) PublisherTool() (*rmqtool.PublisherTool, error) {
	conn := rmqtool.Conn(c.Publisher.Conn)
	if c.Publisher.Key == "" {
		return nil, errors.New("empty publisher key")
	} else if err := conn.CreateExchange(c.Publisher.Exchange); err != nil {
		return nil, err
	} else {
		return conn.ApplyPublisher(c.Publisher.Exchange, []string{c.Publisher.Key})
	}
}

func (c *Config) PublisherKey() string {
	return c.Publisher.Key
}

func LoadConfig(key string) (*Config, error) {
	config := &Config{}
	err := config.Load(key)
	return config, err
}

var logger *zap.Logger

func init() {
	if os.Getenv("DEBUG") == "true" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync()
}

// messageCmd represents the message command
var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "convert uchat message",
	Long:  `convert uchat message`,
	Run: func(cmd *cobra.Command, args []string) {
		rmqtool.DefaultConsumerToolName = viper.GetString("global_consumer_flag")
		// load config
		config, err := LoadConfig("message_config")
		if err != nil {
			logger.Fatal("load config error", zap.Error(err))
		}
		logger.Debug("load message config success", zap.Any("config", config))

		queue, err := config.ConsumerQueue()
		if err != nil {
			logger.Fatal("migrate message_queue error", zap.Error(err))
		}

		publisher, err := config.PublisherTool()
		if err != nil {
			logger.Fatal("migrate message_publisher error", zap.Error(err))
		}

		queue.Consume(1, func(msg amqp.Delivery) {
			ret, err := uchatlib.ConvertUchatMessage(msg.Body)
			if err != nil {
				msg.Ack(false) //先消费掉，避免队列堵塞
				rmqtool.Log.Error("process error", zap.Error(err), zap.Any("msg", msg))
			} else {
				for _, v := range ret {
					b, err := msgpack.Marshal(v)
					if err != nil {
						rmqtool.Log.Error("msgpack error", zap.Error(err))
						continue
					}
					publisher.PublishExt(config.PublisherKey(), FetchMessageRouteFix(v), amqp.Publishing{
						DeliveryMode: amqp.Persistent,
						Timestamp:    time.Now(),
						ContentType:  "application/msgpack",
						Body:         b,
					})
					msg.Ack(false)
				}
			}
		}) //尽量保证聊天记录的时序，以api回调接口收到消息进入receive队列为准
	},
}

func FetchMessageRouteFix(v *uchatlib.UchatMessage) string {
	return fmt.Sprintf(".%s.%d.%s", v.ChatRoomSerialNo, v.MsgType, v.WxUserSerialNo) // .roomid.type.userid
}

func init() {
	RootCmd.AddCommand(messageCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// messageCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// messageCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
