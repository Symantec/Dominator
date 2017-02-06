all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make


dominator.tarball:
	@./scripts/make-tarball dominator -C $(ETCDIR) ssl

filegen-server.tarball:
	@./scripts/make-tarball filegen-server -C $(ETCDIR) ssl

image-unpacker.tarball:
	@./scripts/make-tarball image-unpacker \
		scripts/image-pusher/make-bootable -C $(ETCDIR) ssl

imageserver.tarball:
	@./scripts/make-tarball imageserver -C $(ETCDIR) ssl

mdbd.tarball:
	@./scripts/make-tarball mdbd

subd.tarball:
	@cd c; make
	@./scripts/make-tarball subd -C $(GOPATH) bin/run-in-mntns \
		-C $(ETCDIR) ssl


format:
	gofmt -s -w .


test:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Symantec/Dominator/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test
