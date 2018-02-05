// Copyright © 2018 edwin <edwin.lzh@gmail.com>
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
	"encoding/json"
	"fmt"
	"time"

	rmqtool "github.com/lvzhihao/go-rmqtool"
	"github.com/lvzhihao/uchatlib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

// privateCmd represents the private command
var privateCmd = &cobra.Command{
	Use:   "private",
	Short: "convert uchat message private",
	Long:  `convert uchat message private`,
	Run: func(cmd *cobra.Command, args []string) {
		// zap logger
		logger := GetLogger()
		defer logger.Sync()
		// rmqtool config
		rmqtool.DefaultConsumerToolName = viper.GetString("global_consumer_flag")
		rmqtool.Log.Set(GetZapLoggerWrapperForRmqtool(logger))
		//rmqtool.Log.Debug("logger warpper demo", "no key param", zap.Any("ccc", time.Now()), zap.Any("dddd", []string{"xx"}), "no key param again", zap.Error(errors.New("xx")))
		// load config
		config, err := LoadConfig("private_config")
		if err != nil {
			logger.Fatal("load config error", zap.Error(err))
		}
		logger.Debug("load private config success", zap.Any("config", config))

		queue, err := config.ConsumerQueue()
		if err != nil {
			logger.Fatal("migrate private_queue error", zap.Error(err))
		}

		publisher, err := config.PublisherTool()
		if err != nil {
			logger.Fatal("call private_publisher error", zap.Error(err))
		}

		queue.Consume(1, func(msg amqp.Delivery) {
			ret, err := uchatlib.ConvertUchatMessage(msg.Body)
			if err != nil {
				msg.Ack(false) //先消费掉，避免队列堵塞
				rmqtool.Log.Error("process error", zap.Error(err), zap.Any("msg", msg))
			} else {
				for _, v := range ret {
					b, err := json.Marshal(v)
					if err != nil {
						rmqtool.Log.Error("json marshal error", zap.Error(err))
						continue
					}
					publisher.PublishExt(config.PublisherKey(), FetchPrivateRouteFix(v), amqp.Publishing{
						DeliveryMode: amqp.Persistent,
						Timestamp:    time.Now(),
						ContentType:  "application/json",
						Body:         b,
					})
				}
				msg.Ack(false)
			}
		}) //尽量保证聊天记录的时序，以api回调接口收到消息进入receive队列为准
	},
}

func FetchPrivateRouteFix(v *uchatlib.UchatMessage) string {
	return fmt.Sprintf(".%s.%d.%s", v.RobotSerialNo, v.MsgType, v.WxUserSerialNo) // .roomid.type.userid
}

func init() {
	RootCmd.AddCommand(privateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// privateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// privateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
