package filegenerator

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/mdb"
)

type GetObjectRequest struct {
	Hash hash.Hash
}

type YieldRequest struct {
	Machine   mdb.Machine
	Pathnames []string
}

type ClientRequest struct {
	GetObjectRequest *GetObjectRequest
	YieldRequest     *YieldRequest
}

type GetObjectResponse struct {
	Hash hash.Hash
	Data []byte
}

type FileInfo struct {
	Pathname string
	Hash     hash.Hash
	Length   uint64
}

type YieldResponse struct {
	Hostname string
	Files    []FileInfo
}

// ServerMessage types are sent in response to requests from the client and also
// due to internal state changes in the server.
type ServerMessage struct {
	GetObjectResponse *GetObjectResponse
	YieldResponse     *YieldResponse
}
