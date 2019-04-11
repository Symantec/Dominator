all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make

build-darwin:
	@cd $(GOPATH)/src; (GOOS=darwin go build github.com/Symantec/Dominator/cmd/*)

build-linux:
	@cd $(GOPATH)/src; (GOOS=linux go build github.com/Symantec/Dominator/cmd/*)

build-windows:
	@cd $(GOPATH)/src; (GOOS=windows go build github.com/Symantec/Dominator/cmd/*)

install-darwin:
	@cd $(GOPATH)/src; (GOOS=darwin go install github.com/Symantec/Dominator/cmd/*)

install-linux:
	@cd $(GOPATH)/src; (GOOS=linux go install github.com/Symantec/Dominator/cmd/*)

install-windows:
	@cd $(GOPATH)/src; (GOOS=windows go install github.com/Symantec/Dominator/cmd/*)

dominator.tarball:
	@./scripts/make-tarball dominator -C $(ETCDIR) ssl

filegen-server.tarball:
	@./scripts/make-tarball filegen-server -C $(ETCDIR) ssl

fleet-manager.tarball:
	@./scripts/make-tarball fleet-manager -C $(ETCDIR) ssl

hypervisor.tarball:
	@./scripts/make-tarball hypervisor init.d/virtual-machines.* \
		-C $(ETCDIR) ssl

image-unpacker.tarball:
	@./scripts/make-tarball image-unpacker \
		scripts/image-pusher/export-image -C $(ETCDIR) ssl

installer.tarball:
	@cmd/installer/make-tarball installer -C $(ETCDIR) ssl

imageserver.tarball:
	@./scripts/make-tarball imageserver -C $(ETCDIR) ssl

imaginator.tarball:
	@./scripts/make-tarball imaginator -C $(ETCDIR) ssl

mdbd.tarball:
	@./scripts/make-tarball mdbd

subd.tarball:
	@cd c; make
	@./scripts/make-tarball subd -C $(GOPATH) bin/run-in-mntns \
		-C $(ETCDIR) ssl


format:
	gofmt -s -w .

format-imports:
	goimports -w .


test:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Symantec/Dominator/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test
