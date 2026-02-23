ARG BUILD_FROM
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

ARG TARGETARCH
ARG TARGETVARIANT
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -o /huawei-solar-proxy .

FROM ${BUILD_FROM}

COPY --from=builder /huawei-solar-proxy /usr/local/bin/
CMD ["huawei-solar-proxy", "--ha-options", "/data/options.json"]
