FROM golang:1-alpine as builder

ARG VERSION

RUN go install github.com/korylprince/fileenv@v1.1.0
RUN go install "github.com/korylprince/tcea-inventory-server@$VERSION"


FROM alpine:3.15

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/bin/fileenv /
COPY --from=builder /go/bin/tcea-inventory-server /
COPY setenv.sh /

CMD ["/fileenv", "sh", "/setenv.sh", "/tcea-inventory-server"]
