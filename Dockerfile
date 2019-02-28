FROM golang:alpine AS build

RUN mkdir -p /go/src/github.com/zdnscloud/singlecloud
COPY . /go/src/github.com/zdnscloud/singlecloud

WORKDIR /go/src/github.com/zdnscloud/singlecloud
RUN CGO_ENABLED=0 GOOS=linux go build cmd/singlecloud/singlecloud.go

FROM node:10-alpine AS uibuild

RUN apk --no-cache add ca-certificates git && \
        git clone https://github.com/zdnscloud/singlecloud-ui.git --depth=1 /singlecloud-ui && \
        cd singlecloud-ui && \
        yarn && \
        npx lerna link && \
        cd packages/ui && \
        yarn && \
        yarn run build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /go/src/github.com/zdnscloud/singlecloud/singlecloud /usr/local/bin/
COPY --from=uibuild /singlecloud-ui/packages/ui/build /www

ENTRYPOINT ["singlecloud"]
