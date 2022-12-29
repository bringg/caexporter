FROM golang:1.19-alpine as builder

ARG caexporter_version=0.0.2

RUN apk --no-cache add git

WORKDIR /opt/caexporter
COPY . .
RUN go build -ldflags="-X github.com/prometheus/common/version.Version=${caexporter_version}} -X github.com/prometheus/common/version.Revision=$(git rev-parse HEAD) -X github.com/prometheus/common/version.Branch=$(git branch --show-current)" ./

FROM alpine:3.16
LABEL maintainer "Bringg DevOps <devops@bringg.com>"

USER nobody

COPY --from=builder /opt/caexporter/caexporter /usr/local/bin

ENTRYPOINT ["caexporter"]
