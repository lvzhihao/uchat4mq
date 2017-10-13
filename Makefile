OS := $(shell uname)

build: */*.go
	go build 

receive:
	./uchat4mq receive

migrate: build
	./uchat4mq migrate

docker-build:
	sudo docker build -t edwinlll/uchat4mq:latest .

docker-push:
	sudo docker push edwinlll/uchat4mq:latest

docker-ccr:
	sudo docker tag edwinlll/uchat4mq:latest ccr.ccs.tencentyun.com/wdwd/uchat4mq
	sudo docker push ccr.ccs.tencentyun.com/wdwd/uchat4mq
