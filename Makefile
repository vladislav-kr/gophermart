.PHONY: cover-html
cover-html: 
	go test -v ./... -coverprofile cover.out && go tool cover -html=cover.out && rm cover.out
.PHONY: cover-total
cover-total: 
	go test -v ./... -coverprofile cover.out && go tool cover -func cover.out && rm cover.out
.PHONY: run
run:
	go run ./cmd/gophermart/main.go