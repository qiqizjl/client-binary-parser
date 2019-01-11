FROM golang:alpine AS builder
#增加压缩
RUN apk add upx git;
WORKDIR /go/mobile-parser
COPY . .
# WORKDIR /go/src/github.com/prometheus/client_golang/prometheus
# WORKDIR /go/src/github.com/prometheus/client_golang/examples/simple
RUN  GOOS=linux CGO_ENABLED=0  go build -ldflags "-s -w"  -mod=vendor -o mobile-parser
#RUN  upx mobile-parser

# Final image.
FROM scratch
COPY --from=builder /go/mobile-parser/mobile-parser /
COPY --from=builder /tmp /tmp

EXPOSE 8080
ENTRYPOINT ["/mobile-parser"]
