GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) cmd/singlecloud/singlecloud.go
	go build cmd/singlecloud/singlecloud.go

docker: build-image
	docker push zdnscloud/singlecloud:v0.1.0

build-image:
	rm -rf ui && git clone https://github.com/zdnscloud/singlecloud-ui.git --depth=1 ui
	docker build -t zdnscloud/singlecloud:v0.1.0 .
	docker image prune -f

clean:
	rm -rf singlecloud

.PHONY: clean install
