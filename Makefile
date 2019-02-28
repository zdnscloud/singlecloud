GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) cmd/singlecloud/singlecloud.go
	go build cmd/singlecloud/singlecloud.go

docker:
	docker build -t zdnscloud/singlecloud:v0.1.0 .
	docker image prune -f
	docker push zdnscloud/singlecloud:v0.1.0

clean:
	rm -rf singlecloud

.PHONY: clean install
