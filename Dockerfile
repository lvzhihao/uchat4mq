FROM golang:1.9.1

COPY . /go/src/github.com/lvzhihao/uchat4mq 

WORKDIR /go/src/github.com/lvzhihao/uchat4mq

RUN rm -f /go/src/github.com/lvzhihao/uchat4mq/.uchat4mq.yaml
RUN go-wrapper install
