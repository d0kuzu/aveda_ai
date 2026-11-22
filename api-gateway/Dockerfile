FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY config ./config
COPY grpc ./grpc
COPY middleware ./middleware
COPY proxy ./proxy
COPY routes ./routes
COPY server ./server
COPY proto/db ./proto/db
RUN CGO_ENABLED=0 go build -o /out/api-gateway .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/api-gateway ./
EXPOSE 8080
CMD ["./api-gateway"]
