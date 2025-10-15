# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.version=${VERSION:-dev}" -o /out/docker-agent .

FROM alpine:3.21
WORKDIR /app
COPY --from=build /out/docker-agent /usr/local/bin/docker-agent
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/docker-agent"]
