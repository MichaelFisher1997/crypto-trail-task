.PHONY: test cover coverhtml tidy

PKGS := ./...

test:
	go test $(PKGS) -count=1

cover:
	go test $(PKGS) -count=1 -cover -coverpkg=./...

coverhtml:
	go test $(PKGS) -count=1 -coverpkg=./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html && echo "coverage.html generated"

tidy:
	go mod tidy
