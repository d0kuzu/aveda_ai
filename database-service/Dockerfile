FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY database-service .
COPY migrations ./migrations

RUN chmod +x database-service

EXPOSE 50051

CMD ["./database-service"]
