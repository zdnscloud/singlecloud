VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD}"
GOSRC = $(shell find . -type f -name '*.go')
VERSION=$${VERSION:-dev}

build: singlecloud

singlecloud: $(GOSRC) 
	go build ${LDFLAGS} cmd/singlecloud/singlecloud.go

docker: build-image
	docker push zdnscloud/singlecloud:${VERSION}

build-image:
	docker pull zdnscloud/singlecloud-ui:dev
	docker build -t zdnscloud/singlecloud:${VERSION} .
	docker image prune -f

clean:
	rm -rf singlecloud

.PHONY: clean install
