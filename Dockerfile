
FROM golang:1.18-buster AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd cmd

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -ldflags '-extldflags "-fno-PIC -static"' -buildmode pie -tags 'osusergo netgo static_build' \
    -o server ./cmd/server

FROM gcr.io/distroless/base-debian10 AS runner

WORKDIR /
COPY --from=builder --chown=nonroot:nonroot /app/server /server
USER nonroot:nonroot
CMD ["/server"]
EXPOSE 8080