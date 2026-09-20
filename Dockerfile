# Build Stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /tf-blast ./cmd/tf-blast

# Runtime Stage: Distroless Static Debian 12 (zero-trust, minimal attack surface, non-root)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /work

COPY --from=builder /tf-blast /usr/local/bin/tf-blast

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/tf-blast"]
