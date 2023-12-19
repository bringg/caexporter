FROM golang:1.21-alpine as builder

RUN apk --no-cache add git make

WORKDIR /opt/caexporter
COPY . .
RUN make build

FROM alpine:3.19
LABEL maintainer "Bringg DevOps <devops@bringg.com>"

USER nobody

COPY --from=builder /opt/caexporter/caexporter /usr/local/bin

ENTRYPOINT ["caexporter"]
