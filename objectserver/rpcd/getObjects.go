package rpcd

import (
	"encoding/gob"
	"fmt"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"net/http"
)

func getObjectsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 must CONNECT\n")
		return
	}
	conn, bufrw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		fmt.Println("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	defer conn.Close()
	defer bufrw.Flush()
	io.WriteString(conn, "HTTP/1.0 200 Connected to GetObjects RPC\n\n")
	var request objectserver.GetObjectsRequest
	var response objectserver.GetObjectsResponse
	decoder := gob.NewDecoder(bufrw)
	encoder := gob.NewEncoder(bufrw)
	err = decoder.Decode(&request)
	if err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	response.ObjectSizes, err = objectServer.CheckObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	// First a quick check for existence. If any objects missing, fail request.
	for index, hash := range request.Hashes {
		if response.ObjectSizes[index] < 1 {
			response.ResponseString = fmt.Sprintf("unknown object: %x", hash)
			encoder.Encode(response)
			return
		}
	}
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	encoder.Encode(response)
	bufrw.Flush()
	for range request.Hashes {
		_, reader, err := objectsReader.NextObject()
		if err != nil {
			fmt.Println(err)
			return
		}
		_, err = io.Copy(conn, reader)
		if err != nil {
			fmt.Printf("Error copying:\t%s\n", err)
			return
		}
	}
}
