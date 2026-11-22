FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal ./internal
COPY grpc ./grpc
COPY proto/db ./proto/db
RUN CGO_ENABLED=0 go build -o /out/auth-service .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/auth-service ./
EXPOSE 8080
CMD ["./auth-service"]
