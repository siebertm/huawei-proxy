FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /huawei-solar-proxy .

FROM alpine:3.21
COPY --from=builder /huawei-solar-proxy /usr/local/bin/
ENTRYPOINT ["huawei-solar-proxy"]
CMD ["-config", "/etc/huawei-solar-proxy/config.yaml"]
