FROM golang:latest as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest  
ENV GO_APP_ENV local
ARG APP_VERSION local
ENV APP_VERSION $APP_VERSION
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY migrations ./migrations

EXPOSE 8080

CMD ["./main"] 