FROM golang:latest

WORKDIR /app
COPY . .

RUN go mod tidy && go build -o app ./cmd/raft/main.go

EXPOSE 8080
CMD ["./app"]
