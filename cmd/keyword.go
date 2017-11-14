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
	"time"

	uchat4mq "github.com/lvzhihao/uchat4mq/libs"
	"github.com/lvzhihao/uchatlib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

var (
	publisher *uchat4mq.PublisherTool
)

// keywordCmd represents the keyword command
var keywordCmd = &cobra.Command{
	Use:   "keyword",
	Short: "convert uchat keyword",
	Long:  `convert uchat keyword`,
	Run: func(cmd *cobra.Command, args []string) {
		// zap logger
		logger := GetLogger()
		defer logger.Sync()

		//migrate msginfo exchange
		if err := uchat4mq.CreateExchange(
			viper.GetString("rabbitmq_api"),
			viper.GetString("rabbitmq_user"),
			viper.GetString("rabbitmq_passwd"),
			viper.GetString("rabbitmq_vhost"),
			viper.GetString("rabbitmq_msginfo_exchange_name"),
		); err != nil {
			logger.Fatal("migrate msginfo_exchange error", zap.Error(err))
		}

		//migrate 4pre message for receive exchange
		if err := uchat4mq.RegisterQueue(
			viper.GetString("rabbitmq_api"),
			viper.GetString("rabbitmq_user"),
			viper.GetString("rabbitmq_passwd"),
			viper.GetString("rabbitmq_vhost"),
			viper.GetString("uchat_receive_keyword_queue_name"),
			viper.GetString("rabbitmq_receive_exchange_name"),
			viper.GetStringSlice("uchat_receive_keyword_queue_keys"),
		); err != nil {
			logger.Fatal("migrate keyword_queue error", zap.Error(err))
		}

		var err error
		publisher, err = uchat4mq.NewPublisherTool(
			fmt.Sprintf("amqp://%s:%s@%s/%s", viper.GetString("rabbitmq_user"), viper.GetString("rabbitmq_passwd"), viper.GetString("rabbitmq_host"), viper.GetString("rabbitmq_vhost")),
			viper.GetString("rabbitmq_msginfo_exchange_name"),
			[]string{"uchat.process.keyword"},
			logger,
		)
		if err != nil {
			logger.Fatal("publisher create error", zap.Error(err))
		}

		consumer, err := uchat4mq.NewConsumerTool(
			fmt.Sprintf("amqp://%s:%s@%s/%s", viper.GetString("rabbitmq_user"), viper.GetString("rabbitmq_passwd"), viper.GetString("rabbitmq_host"), viper.GetString("rabbitmq_vhost")),
			logger,
		)
		if err != nil {
			logger.Fatal("consumer create error", zap.Error(err))
		}
		consumer.Consume(viper.GetString("uchat_receive_keyword_queue_name"), 1, processKeyword) //尽量保证聊天记录的时序，以api回调接口收到消息进入receive队列为准
	},
}

func FetchKeywordRouteFix(v *uchatlib.UchatKeyword) string {
	return fmt.Sprintf(".%s.%s.%s", v.ChatRoomSerialNo, v.FromWxUserSerialNo, v.ToWxUserSerialNo) // .roomid.type.userid
}

func processKeyword(msg amqp.Delivery, logger *zap.Logger) {
	ret, err := uchatlib.ConvertUchatKeyword(msg.Body)
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
			publisher.PublishExt("uchat.process.keyword", FetchKeywordRouteFix(v), amqp.Publishing{
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
	RootCmd.AddCommand(keywordCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// keywordCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// keywordCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
