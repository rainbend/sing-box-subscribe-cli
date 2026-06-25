FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN set -eux; \
    goos="${TARGETOS:-linux}"; \
    goarch="${TARGETARCH:-$(go env GOARCH)}"; \
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/sing-box-sub \
      ./cmd/sing-box-subscribe-cli

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/sing-box-sub /usr/local/bin/sing-box-sub

WORKDIR /work
ENTRYPOINT ["/usr/local/bin/sing-box-sub"]
