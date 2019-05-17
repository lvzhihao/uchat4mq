OS := $(shell uname)

build: */*.go
	go build 

dev:
	DEBUG=true go run main.go message

message:
	./uchat4mq message

migrate: build
	./uchat4mq migrate

docker-build:
	 docker build -t edwinlll/uchat4mq:latest .

docker-push:
	 docker push edwinlll/uchat4mq:latest

docker-ccr:
	 docker tag edwinlll/uchat4mq:latest ccr.ccs.tencentyun.com/wdwd/uchat4mq:latest
	 docker push ccr.ccs.tencentyun.com/wdwd/uchat4mq:latest
	 docker rmi ccr.ccs.tencentyun.com/wdwd/uchat4mq:latest

docker-uhub:
	 docker tag edwinlll/uchat4mq:latest uhub.service.ucloud.cn/mmzs/uchat4mq:latest
	 docker push uhub.service.ucloud.cn/mmzs/uchat4mq:latest
	 docker rmi uhub.service.ucloud.cn/mmzs/uchat4mq:latest

docker-ali:
	 docker tag edwinlll/uchat4mq:latest registry.cn-hangzhou.aliyuncs.com/weishangye/uchat4mq:latest
	 docker push registry.cn-hangzhou.aliyuncs.com/weishangye/uchat4mq:latest
	 docker rmi registry.cn-hangzhou.aliyuncs.com/weishangye/uchat4mq:latest
