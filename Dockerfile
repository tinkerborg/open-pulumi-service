ARG GO_VERSION=1.25

FROM golang:${GO_VERSION} AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o open-pulumi ./cmd/server

FROM scratch

COPY --from=builder /build/open-pulumi .

ENV LISTEN_ADDRESS=0.0.0.0 LISTEN_PORT=8080
EXPOSE 8080

ENTRYPOINT ["./open-pulumi"]
