build:
	@go build -o bin/taller-ing-de-software-backend ./cmd/main.go

test:
	@go test -v ./handlers/tests
	
run: build
	@./bin/taller-ing-de-software-backend