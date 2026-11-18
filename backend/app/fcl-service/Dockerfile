FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/fcl-service .

FROM alpine:3.20
RUN adduser -D -H app
COPY --from=builder /out/fcl-service /usr/local/bin/fcl-service
USER app
EXPOSE 6905
CMD ["fcl-service"]
