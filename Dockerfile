FROM golang:1.9

WORKDIR /go/src/github.com/lvzhihao/uchat4mq

COPY . . 

RUN rm -f /go/src/github.com/lvzhihao/uchat4mq/.uchat4mq.yaml
RUN go-wrapper install
