FROM golang:1.24-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sing-box-sub ./cmd/sing-box-subscribe-cli

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/sing-box-sub /usr/local/bin/sing-box-sub

WORKDIR /work
ENTRYPOINT ["/usr/local/bin/sing-box-sub"]
