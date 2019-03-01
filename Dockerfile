FROM golang:alpine AS build

RUN mkdir -p /go/src/github.com/zdnscloud/singlecloud
COPY . /go/src/github.com/zdnscloud/singlecloud

WORKDIR /go/src/github.com/zdnscloud/singlecloud
RUN CGO_ENABLED=0 GOOS=linux go build cmd/singlecloud/singlecloud.go

FROM node:10-alpine AS uibuild

COPY ./ui /ui
RUN cd /ui && \
        npx lerna bootstrap && \
        cd packages/ui && \
        yarn run build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /go/src/github.com/zdnscloud/singlecloud/singlecloud /usr/local/bin/
COPY --from=uibuild /ui/packages/ui/build /www

ENTRYPOINT ["singlecloud"]
