all:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/*
	@cd c; make


DOMINATOR_TARGET = /tmp/$(LOGNAME)/dominator.tar.gz
FILEGEN_SERVER_TARGET = /tmp/$(LOGNAME)/filegen-server.tar.gz
IMAGE_UNPACKER_TARGET = /tmp/$(LOGNAME)/image-unpacker.tar.gz
IMAGESERVER_TARGET = /tmp/$(LOGNAME)/imageserver.tar.gz
MDBD_TARGET = /tmp/$(LOGNAME)/mdbd.tar.gz
SUBD_TARGET = /tmp/$(LOGNAME)/subd.tar.gz

dominator.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/dominator
	@tar --owner=0 --group=0 -czf $(DOMINATOR_TARGET) \
	init.d/dominator.* \
	scripts/install.lib \
	-C cmd/dominator install \
	-C $(GOPATH) bin/dominator \
	-C $(ETCDIR) ssl

filegen-server.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/filegen-server
	@tar --owner=0 --group=0 -czf $(FILEGEN_SERVER_TARGET) \
	init.d/filegen-server.* \
	scripts/install.lib \
	-C cmd/filegen-server install \
	-C $(GOPATH) bin/filegen-server \
	-C $(ETCDIR) ssl

image-unpacker.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/image-unpacker
	@tar --owner=0 --group=0 -czf $(IMAGE_UNPACKER_TARGET) \
	init.d/image-unpacker.* \
	scripts/install.lib \
	scripts/image-pusher/make-bootable \
	-C cmd/image-unpacker install \
	-C $(GOPATH) bin/image-unpacker \
	-C $(ETCDIR) ssl

imageserver.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/imageserver
	@tar --owner=0 --group=0 -czf $(IMAGESERVER_TARGET) \
	init.d/imageserver.* \
	scripts/install.lib \
	-C cmd/imageserver install \
	-C $(GOPATH) bin/imageserver \
	-C $(ETCDIR) ssl

mdbd.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/mdbd
	@tar --owner=0 --group=0 -czf $(MDBD_TARGET) \
	init.d/mdbd.* \
	scripts/install.lib \
	-C cmd/mdbd install \
	-C $(GOPATH) bin/mdbd

subd.tarball:
	@cd $(GOPATH)/src; go install github.com/Symantec/Dominator/cmd/subd
	@cd c; make
	@tar --owner=0 --group=0 -czf $(SUBD_TARGET) \
	init.d/subd.* \
	scripts/install.lib \
	-C cmd/subd install \
	-C $(GOPATH) bin/run-in-mntns bin/subd \
	-C $(ETCDIR) ssl


format:
	gofmt -s -w .


test:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Symantec/Dominator/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test
