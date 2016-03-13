package filegenerator

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"time"
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
	Pathname   string
	Hash       hash.Hash
	ValidUntil time.Time
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
