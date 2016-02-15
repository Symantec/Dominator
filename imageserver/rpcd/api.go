package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/srpc"
	"io"
	"log"
	"net/rpc"
	"sync"
)

type rpcType struct {
	imageDataBase *scanner.ImageDataBase
	logger        *log.Logger
}

type srpcType struct {
	imageDataBase             *scanner.ImageDataBase
	replicationMaster         string
	logger                    *log.Logger
	numReplicationClientsLock sync.RWMutex // Protect numReplicationClients.
	numReplicationClients     uint
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

var replicationMessage = "cannot make changes while under replication control" +
	", go to master: "

func Setup(imdb *scanner.ImageDataBase, replicationMaster string,
	lg *log.Logger) *htmlWriter {
	rpcObj := rpcType{
		imageDataBase: imdb,
		logger:        lg}
	rpc.RegisterName("ImageServer", &rpcObj)
	srpcObj := srpcType{
		imageDataBase:     imdb,
		replicationMaster: replicationMaster,
		logger:            lg}
	srpc.RegisterName("ImageServer", &srpcObj)
	return (*htmlWriter)(&srpcObj)
}
