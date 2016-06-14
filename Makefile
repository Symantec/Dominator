all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make


SUBD_TARGET = /tmp/$(LOGNAME)/subd.tar.gz

subd.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/subd
	@cd c; make
	@tar --owner=0 --group=0 -czf $(SUBD_TARGET) \
	init.d/subd.* \
	scripts/install.lib \
	-C sub install \
	-C $(GOPATH) bin/run-in-mntns bin/subd \
	-C $(ETCDIR) ssl


format:
	gofmt -s -w .


test:
	@find * -name '*_test.go' -printf 'github.com/Symantec/Dominator/%h\n' \
	| sort -u | xargs -r go test
