FROM golang:1.11-alpine as builder

ARG VERSION

RUN apk add --no-cache git ca-certificates

RUN git clone --branch "v1.1" --single-branch --depth 1 \
    https://github.com/korylprince/fileenv.git /go/src/github.com/korylprince/fileenv

RUN git clone --branch "$VERSION" --single-branch --depth 1 \
    https://github.com/korylprince/tcea-inventory-server.git  /go/src/github.com/korylprince/tcea-inventory-server

RUN go install github.com/korylprince/fileenv
RUN go install github.com/korylprince/tcea-inventory-server

FROM alpine:3.8

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/bin/fileenv /
COPY --from=builder /go/bin/tcea-inventory-server /
COPY setenv.sh /

CMD ["/fileenv", "sh", "/setenv.sh", "/tcea-inventory-server"]

