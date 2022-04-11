.PHONY : lint format coverage unit integration
LINT_SETTINGS=golint,misspell,gocyclo,gocritic,whitespace,goconst,bodyclose,unconvert,lll

all: lint format build

format:
	gofmt -s -w .
	goimports -w .

lint:
	golangci-lint run --timeout 2m0s -v -E ${LINT_SETTINGS}

unit:
	go test -tags="relic" -v ./...

integration:
	go test -v -tags="relic integration" ./...

rightparen:=)
coverage:
	mkdir -p coverage
	go test -tags="relic integration" -v -coverpkg=./... -coverprofile=profileTMP  ./...

	cat profileTMP	| grep -v "main.go"		>	profileTMP1
	cat profileTMP1 | grep -v "flow-rosetta/testing/mocks/" > ./c.out
	rm profileTMP
	rm profileTMP1
	
	go tool cover -func=./c.out
	go tool cover -html=./c.out -o ./coverage/coverage.html
	chmod +x "./scripts/coverage-check.sh"
	./scripts/coverage-check.sh 60 $$(echo $$(go tool cover -func ./c.out) | cut -d '$(rightparen)' -f 2)


