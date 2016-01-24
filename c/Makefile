all: $(GOPATH)/bin/run-in-mntns

$(GOPATH)/bin/run-in-mntns: run-in-mntns.c
	cc -static -s -o $(GOPATH)/bin/run-in-mntns run-in-mntns.c
