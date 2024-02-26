FROM golang:1.20-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o prometheus-broadcaster .
FROM alpine:latest
WORKDIR /app
COPY --from=build /app/prometheus-broadcaster .
COPY --from=build /app/config/config.yaml .
RUN ls
EXPOSE 8080
CMD ["./prometheus-broadcaster"]
