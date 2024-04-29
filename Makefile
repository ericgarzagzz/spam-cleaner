run:
	go run main.go messages.go

build:
	go build -o bin/spam-cleaner main.go messages.go
