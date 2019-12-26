FROM golang:1.12.5-alpine3.9 AS build

ARG version
ARG buildtime

RUN mkdir -p /go/src/github.com/zdnscloud/singlecloud
COPY . /go/src/github.com/zdnscloud/singlecloud
WORKDIR /go/src/github.com/zdnscloud/singlecloud
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime" cmd/singlecloud/singlecloud.go
RUN apk add curl
RUN curl -L https://raw.githubusercontent.com/zdnscloud/zke/split-image-config/default_image.yml -o /default_image.yml
    

FROM scratch
COPY --from=build /go/src/github.com/zdnscloud/singlecloud/singlecloud /
COPY --from=build /default_image.yml /
ENTRYPOINT ["/singlecloud"]
