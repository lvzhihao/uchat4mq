FROM golang:1.9

WORKDIR /go/src/github.com/lvzhihao/uchat4mq

COPY . . 

RUN go get github.com/yanyiwu/gojieba

RUN go-wrapper install && \
    rm -rf *
