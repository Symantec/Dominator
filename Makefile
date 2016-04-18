all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make

format:
	gofmt -s -w .

test:
	@find * -name '*_test.go' -printf 'github.com/Symantec/Dominator/%h\n' \
	| sort -u | xargs -r go test
