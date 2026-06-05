FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /bin/backctl-mcp ./cmd/backctl-mcp

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /bin/backctl ./cmd/backctl

FROM alpine:3.23

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/backctl-mcp /usr/local/bin/backctl-mcp
COPY --from=builder /bin/backctl /usr/local/bin/backctl

ENV BACKSTAGE_URL=""
ENV BACKSTAGE_TOKEN=""

ENTRYPOINT ["backctl-mcp"]
