all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make

format:
	gofmt -s -w .
