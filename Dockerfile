FROM golang:1.13.7-alpine3.11 AS build
ENV GOPROXY=https://goproxy.cn

ARG version
ARG buildtime

RUN mkdir -p /go/src/github.com/zdnscloud/singlecloud
COPY . /go/src/github.com/zdnscloud/singlecloud
WORKDIR /go/src/github.com/zdnscloud/singlecloud

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime -X github.com/zdnscloud/singlecloud/pkg/zke.singleCloudVersion=$version -X 'github.com/zdnscloud/singlecloud/vendor/github.com/zdnscloud/zke/types.imageConfig=`cat zke_image.yml`'" cmd/singlecloud/singlecloud.go


FROM scratch
COPY --from=build /go/src/github.com/zdnscloud/singlecloud/singlecloud /
ENTRYPOINT ["/singlecloud"]
