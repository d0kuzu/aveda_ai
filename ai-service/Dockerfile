FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY ai-service .
COPY json ./json

RUN chmod +x ai-service

EXPOSE 8080

CMD ["./ai-service"]
