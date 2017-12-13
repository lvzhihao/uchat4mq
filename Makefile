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
	sudo docker build -t edwinlll/uchat4mq:latest .

docker-push:
	sudo docker push edwinlll/uchat4mq:latest

docker-ccr:
	sudo docker tag edwinlll/uchat4mq:latest ccr.ccs.tencentyun.com/wdwd/uchat4mq:latest
	sudo docker push ccr.ccs.tencentyun.com/wdwd/uchat4mq:latest

docker-uhub:
	sudo docker tag edwinlll/uchat4mq:latest uhub.service.ucloud.cn/mmzs/uchat4mq:latest
	sudo docker push uhub.service.ucloud.cn/mmzs/uchat4mq:latest
