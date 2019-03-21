GOSRC = $(shell find . -type f -name '*.go')
VERSION=$${VERSION:-dev}

build: singlecloud

singlecloud: $(GOSRC) cmd/singlecloud/singlecloud.go
	go build cmd/singlecloud/singlecloud.go

docker: build-image
	docker push zdnscloud/singlecloud:${VERSION}

build-image:
	docker pull zdnscloud/singlecloud-ui:dev
	docker build -t zdnscloud/singlecloud:${VERSION} .
	docker image prune -f

clean:
	rm -rf singlecloud

.PHONY: clean install
