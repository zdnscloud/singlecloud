VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`
BRANCH=`git branch | sed -n '/\* /s///p'`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD}"
GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) 
	cp zke_image.yml vendor/github.com/zdnscloud/zke/image_config.yml
	go generate vendor/github.com/zdnscloud/zke/types/generate.go
	go build ${LDFLAGS} cmd/singlecloud/singlecloud.go
	rm -f vendor/github.com/zdnscloud/zke/image_config.yml
	rm -f vendor/github.com/zdnscloud/zke/types/initer.go

docker: build-image
	docker push zdnscloud/singlecloud:${BRANCH}
	#docker tag zdnscloud/singlecloud:${VERSION} zdnscloud/singlecloud:latest
	#docker push zdnscloud/singlecloud:latest

build-image:
	docker build -t zdnscloud/singlecloud:${BRANCH} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --no-cache .
	docker image prune -f

clean:
	rm -rf singlecloud

clean-image:
	docker rmi zdnscloud/singlecloud:${VERSION}

.PHONY: clean install
