package rpcd

import (
	"errors"
	"flag"
	"io"
	"sync"

	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
)

var (
	archiveExpiringImages = flag.Bool("archiveExpiringImages", false,
		"If true, replicate expiring images when in archive mode")
	archiveMode = flag.Bool("archiveMode", false,
		"If true, disable delete operations and require update server")
)

type srpcType struct {
	imageDataBase             *scanner.ImageDataBase
	replicationMaster         string
	imageserverResource       *srpc.ClientResource
	objSrv                    objectserver.FullObjectServer
	archiveMode               bool
	logger                    log.Logger
	numReplicationClientsLock sync.RWMutex // Protect numReplicationClients.
	numReplicationClients     uint
	imagesBeingInjectedLock   sync.Mutex // Protect imagesBeingInjected.
	imagesBeingInjected       map[string]struct{}
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

var replicationMessage = "cannot make changes while under replication control" +
	", go to master: "

func Setup(imdb *scanner.ImageDataBase, replicationMaster string,
	objSrv objectserver.FullObjectServer,
	logger log.Logger) (*htmlWriter, error) {
	if *archiveMode && replicationMaster == "" {
		return nil, errors.New("replication master required in archive mode")
	}
	srpcObj := &srpcType{
		imageDataBase:       imdb,
		replicationMaster:   replicationMaster,
		imageserverResource: srpc.NewClientResource("tcp", replicationMaster),
		objSrv:              objSrv,
		logger:              logger,
		archiveMode:         *archiveMode,
		imagesBeingInjected: make(map[string]struct{}),
	}
	srpc.RegisterNameWithOptions("ImageServer", srpcObj, srpc.ReceiverOptions{
		PublicMethods: []string{
			"CheckDirectory",
			"CheckImage",
			"FindLatestImage",
			"GetImage",
			"ListDirectories",
			"ListImages",
		}})
	if replicationMaster != "" {
		go srpcObj.replicator()
	}

	return (*htmlWriter)(srpcObj), nil
}
