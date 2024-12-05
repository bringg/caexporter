FROM golang:1.23-alpine as builder

RUN apk --no-cache add git make

WORKDIR /opt/caexporter
COPY . .
RUN make build


FROM gcr.io/distroless/static
LABEL maintainer "Bringg DevOps <devops@bringg.com>"

USER nobody

COPY --from=builder /opt/caexporter/caexporter /

ENTRYPOINT ["/caexporter"]
