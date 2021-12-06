FROM golang:1.17-alpine as builder

ARG caexporter_version=0.0.1

RUN apk --no-cache add git

WORKDIR /opt/caexporter
COPY . .
RUN go build -ldflags="-X github.com/prometheus/common/version.Version=${caexporter_version}} -X github.com/prometheus/common/version.Revision=$(git rev-parse HEAD) -X github.com/prometheus/common/version.Branch=$(git branch --show-current)" ./

FROM alpine:3.15
LABEL maintainer "Bringg DevOps <devops@bringg.com>"

COPY --from=builder /opt/caexporter/caexporter /usr/local/bin

ENTRYPOINT ["caexporter"]
