FROM golang:1.17-alpine as builder

WORKDIR /opt/caexporter
COPY . .
RUN go build

FROM alpine:3.15
LABEL maintainer "Bringg DevOps <devops@bringg.com>"

COPY --from=builder /opt/caexporter/caexporter /usr/local/bin

ENTRYPOINT ["caexporter"]
