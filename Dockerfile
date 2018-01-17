FROM golang:1.9-alpine as builder
WORKDIR /go/src/github.com/lvzhihao/uchat4mq
COPY . . 
RUN apk add --update gcc g++ git && \
    CGO_ENABLED=1 GOOS=linux go build -a .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates tzdata libgcc libstdc++
WORKDIR /usr/local/uchat4mq
COPY --from=builder /go/src/github.com/lvzhihao/uchat4mq/uchat4mq .
COPY --from=builder /go/src/github.com/lvzhihao/uchat4mq/vendor/github.com/yanyiwu/gojieba/dict /go/src/github.com/lvzhihao/uchat4mq/vendor/github.com/yanyiwu/gojieba/dict
ENV PATH /usr/local/uchat4mq:$PATH
