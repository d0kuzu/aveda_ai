FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY ai-service .

RUN chmod +x ai-service

EXPOSE 8080

CMD ["./ai-service"]
