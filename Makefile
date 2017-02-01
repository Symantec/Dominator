all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make


IMAGE_UNPACKER_TARGET = /tmp/$(LOGNAME)/image-unpacker.tar.gz
IMAGESERVER_TARGET = /tmp/$(LOGNAME)/imageserver.tar.gz
SUBD_TARGET = /tmp/$(LOGNAME)/subd.tar.gz

image-unpacker.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/image-unpacker
	@tar --owner=0 --group=0 -czf $(IMAGE_UNPACKER_TARGET) \
	init.d/image-unpacker.* \
	scripts/install.lib \
	scripts/image-pusher/make-bootable \
	-C imageunpacker install \
	-C $(GOPATH) bin/image-unpacker \
	-C $(ETCDIR) ssl

imageserver.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/imageserver
	@tar --owner=0 --group=0 -czf $(IMAGESERVER_TARGET) \
	init.d/imageserver.* \
	scripts/install.lib \
	-C imageserver install \
	-C $(GOPATH) bin/imageserver \
	-C $(ETCDIR) ssl

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
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Symantec/Dominator/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test
