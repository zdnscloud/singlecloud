GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) cmd/singlecloud/singlecloud.go
	go build cmd/singlecloud/singlecloud.go

docker:
	docker build -t zdnscloud/singlecloud:latest .
	docker image prune -f
	docker push zdnscloud/singlecloud:latest

clean:
	rm -rf singlecloud

.PHONY: clean install
