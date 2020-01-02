VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`
BRANCH=`git branch | sed -n '/\* /s///p'`
IMAGE_INITER_FILE=vendor/github.com/zdnscloud/zke/types/initer.go

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD} -X github.com/zdnscloud/singlecloud/pkg/zke.singleCloudVersion=${VERSION}"
GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) 
	if [ -f $(IMAGE_INITER_FILE) ]; then rm $(IMAGE_INITER_FILE); fi
	cp zke_image.yml vendor/github.com/zdnscloud/zke/image_config.yml
	go generate vendor/github.com/zdnscloud/zke/types/generate.go
	go build ${LDFLAGS} cmd/singlecloud/singlecloud.go
	rm -f vendor/github.com/zdnscloud/zke/image_config.yml
	rm -f ${IMAGE_INITER_FILE}

docker: build-image
	docker push zdnscloud/singlecloud:${BRANCH}

build-image:
	docker build -t zdnscloud/singlecloud:${BRANCH} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --no-cache .
	docker image prune -f

clean:
	rm -rf singlecloud

clean-image:
	docker rmi zdnscloud/singlecloud:${VERSION}

.PHONY: clean install
