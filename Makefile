GOSRC = $(shell find . -type f -name '*.go')
VERSION=$${VERSION:-dev}

build: singlecloud

singlecloud: $(GOSRC) cmd/singlecloud/singlecloud.go
	go build cmd/singlecloud/singlecloud.go

docker: build-image
	docker push zdnscloud/singlecloud:${VERSION}

build-image:
	rm -rf ui && git clone https://github.com/zdnscloud/singlecloud-ui.git --depth=1 ui
	docker build -t zdnscloud/singlecloud:${VERSION} .
	docker image prune -f

clean:
	rm -rf singlecloud

.PHONY: clean install
