FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal ./internal
COPY proto/db ./proto/db
RUN CGO_ENABLED=0 go build -o /out/ai-service .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/ai-service ./
EXPOSE 8080
CMD ["./ai-service"]
