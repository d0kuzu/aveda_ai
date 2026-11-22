FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY models ./models
COPY repository ./repository
COPY server ./server
COPY proto ./proto
RUN CGO_ENABLED=0 go build -o /out/database-service .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/database-service ./
COPY migrations ./migrations
EXPOSE 50051
CMD ["./database-service"]
