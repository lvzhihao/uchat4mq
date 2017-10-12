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
	"fmt"
	"os"
	"time"

	"github.com/lvzhihao/uchat4mq/libs"
	"github.com/lvzhihao/uchatlib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

var (
	publisher *libs.PublisherTool
)

// messageCmd represents the message command
var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "convert uchat message",
	Long:  `convert uchat message`,
	Run: func(cmd *cobra.Command, args []string) {
		// zap logger
		var logger *zap.Logger
		if os.Getenv("DEBUG") == "true" {
			logger, _ = zap.NewDevelopment()
		} else {
			logger, _ = zap.NewProduction()
		}
		defer logger.Sync()

		var err error
		publisher, err = libs.NewPublisherTool(
			fmt.Sprintf("amqp://%s:%s@%s/%s", viper.GetString("rabbitmq_user"), viper.GetString("rabbitmq_passwd"), viper.GetString("rabbitmq_host"), viper.GetString("rabbitmq_vhost")),
			viper.GetString("rabbitmq_msginfo_exchange_name"),
			[]string{"uchat.process.message"},
			logger,
		)
		if err != nil {
			logger.Fatal("publisher create error", zap.Error(err))
		}

		consumer, err := libs.NewConsumerTool(
			fmt.Sprintf("amqp://%s:%s@%s/%s", viper.GetString("rabbitmq_user"), viper.GetString("rabbitmq_passwd"), viper.GetString("rabbitmq_host"), viper.GetString("rabbitmq_vhost")),
			logger,
		)
		if err != nil {
			logger.Fatal("consumer create error", zap.Error(err))
		}
		consumer.Consume(viper.GetString("uchat_receive_message_queue"), 1, processMessage) //尽量保证聊天记录的时序，以api回调接口收到消息进入receive队列为准
	},
}

func FetchRouteFix(v *uchatlib.UchatMessage) string {
	return fmt.Sprintf(".%s.%d.%s", v.ChatRoomSerialNo, v.MsgType, v.WxUserSerialNo) // .roomid.type.userid
}

func processMessage(msg amqp.Delivery, logger *zap.Logger) {
	ret, err := uchatlib.ConvertUchatMessage(msg.Body)
	if err != nil {
		msg.Ack(false) //先消费掉，避免队列堵塞
		logger.Error("process error", zap.Error(err), zap.Any("msg", msg))
	} else {
		for _, v := range ret {
			b, err := msgpack.Marshal(v)
			if err != nil {
				logger.Error("msgpack error", zap.Error(err))
				continue
			}
			publisher.PublishExt("uchat.process.message", FetchRouteFix(v), amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				Timestamp:    time.Now(),
				ContentType:  "application/msgpack",
				Body:         b,
			})
			msg.Ack(false)
		}
	}
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
