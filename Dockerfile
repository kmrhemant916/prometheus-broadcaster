FROM golang:1.17-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o prometheus-broadcaster .
FROM alpine:latest
WORKDIR /app
COPY --from=build /app/prometheus-broadcaster .
EXPOSE 8080
CMD ["./prometheus-broadcaster"]
