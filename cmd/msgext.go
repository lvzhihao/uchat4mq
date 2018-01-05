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
	"strings"
	"time"
	"unicode/utf8"

	rmqtool "github.com/lvzhihao/go-rmqtool"
	"github.com/lvzhihao/uchatlib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/yanyiwu/gojieba"
	"go.uber.org/zap"
)

type UchatMsgExt struct {
	uchatlib.UchatMessage
	ContentLength     int //内容长度
	SegmentationCount int //内容分词数量
	SegmentationNo    int
	SegmentationWord  string //单个分词，统计词云使用
	SegmentationType  string //单个分词词性，统计词云使用
}

// msgextCmd represents the msgext command
var msgextCmd = &cobra.Command{
	Use:   "msgext",
	Short: "msgext",
	Long:  `msgext`,
	Run: func(cmd *cobra.Command, args []string) {
		// zap logger
		logger := GetLogger()
		defer logger.Sync()
		// rmqtool config
		rmqtool.DefaultConsumerToolName = viper.GetString("global_consumer_flag")
		rmqtool.Log.Set(GetZapLoggerWrapperForRmqtool(logger))
		//rmqtool.Log.Debug("logger warpper demo", "no key param", zap.Any("ccc", time.Now()), zap.Any("dddd", []string{"xx"}), "no key param again", zap.Error(errors.New("xx")))
		// load config
		config, err := LoadConfig("msgext_config")
		if err != nil {
			logger.Fatal("load config error", zap.Error(err))
		}
		logger.Debug("load msgext config success", zap.Any("config", config))

		queue, err := config.ConsumerQueue()
		if err != nil {
			logger.Fatal("migrate msgext_queue error", zap.Error(err))
		}

		publisher, err := config.PublisherTool()
		if err != nil {
			logger.Fatal("call msgext_publisher error", zap.Error(err))
		}

		x := gojieba.NewJieba()
		defer x.Free()

		queue.Consume(1, func(msg amqp.Delivery) {
			var rst UchatMsgExt
			err := json.Unmarshal(msg.Body, &rst)
			if err != nil {
				msg.Ack(false) //先消费掉，避免队列堵塞
				rmqtool.Log.Error("process error", zap.Error(err), zap.Any("msg", msg))
			} else {
				// 如果是文本信息，则进行分词处理
				if rst.MsgType == 2001 {
					words := x.Tag(strings.TrimSpace(rst.Content))
					rst.ContentLength = utf8.RuneCountInString(rst.Content)
					rst.SegmentationCount = len(words)
					lenB, _ := json.Marshal(rst)
					publisher.PublishExt(config.PublisherKey(), ".length"+FetchMsgextRouteFix(&rst), amqp.Publishing{
						DeliveryMode: amqp.Persistent,
						Timestamp:    time.Now(),
						ContentType:  "application/json",
						Body:         lenB,
					})
					for no, word := range words {
						info := strings.Split(word, "/")
						rst.SegmentationNo = no
						rst.SegmentationWord = info[0]
						rst.SegmentationType = info[1]
						lenW, _ := json.Marshal(rst)
						publisher.PublishExt(config.PublisherKey(), ".words"+FetchMsgextRouteFix(&rst), amqp.Publishing{
							DeliveryMode: amqp.Persistent,
							Timestamp:    time.Now(),
							ContentType:  "application/json",
							Body:         lenW,
						})
					}
				}
				msg.Ack(false)
			}
		}) //尽量保证聊天记录的时序，以api回调接口收到消息进入receive队列为准
	},
}

func FetchMsgextRouteFix(v *UchatMsgExt) string {
	return fmt.Sprintf(".%s.%d.%s", v.ChatRoomSerialNo, v.MsgType, v.WxUserSerialNo) // .roomid.type.userid
}

func init() {
	RootCmd.AddCommand(msgextCmd)
}
