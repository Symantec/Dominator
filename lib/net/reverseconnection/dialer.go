package reverseconnection

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/nulllogger"
)

func newDialer(rawDialer *net.Dialer, serveMux *http.ServeMux,
	minimumInterval, maximumInterval time.Duration,
	logger log.DebugLogger) *Dialer {
	if rawDialer == nil {
		rawDialer = &net.Dialer{}
	}
	if serveMux == nil {
		serveMux = http.DefaultServeMux
	}
	if minimumInterval < time.Second {
		minimumInterval = time.Second
	}
	if maximumInterval <= minimumInterval {
		maximumInterval = 0
	}
	if logger == nil {
		logger = nulllogger.New()
	}
	dialer := &Dialer{
		dialer:          rawDialer,
		minimumInterval: minimumInterval,
		maximumInterval: maximumInterval,
		logger:          logger,
		connectionMap:   make(map[string]net.Conn),
	}
	serveMux.HandleFunc(urlPath, dialer.connectHandler)
	return dialer
}

// Add a connection to the map. Returns true if added, false if duplicate.
func (d *Dialer) add(address string, conn net.Conn) bool {
	d.connectionMapLock.Lock()
	defer d.connectionMapLock.Unlock()
	if _, ok := d.connectionMap[address]; ok {
		return false
	} else {
		d.connectionMap[address] = conn
		return true
	}
}

func (d *Dialer) dial(network, address string) (net.Conn, error) {
	if network != "tcp" || len(d.connectionMap) < 1 {
		return d.dialer.Dial(network, address)
	}
	if conn, err := d.lookupDial(address); err != nil {
		return nil, err
	} else if conn != nil {
		return conn, nil
	}
	return d.dialer.Dial(network, address)
}

func (d *Dialer) lookupDial(address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	if len(addrs) < 1 {
		return nil, nil
	}
	for _, addr := range addrs {
		oneAddress := net.JoinHostPort(addr, port)
		if conn := d.lookup(oneAddress); conn != nil {
			d.logger.Debugf(0, "Consuming reverse dialer connection from: %s\n",
				oneAddress)
			// Tell other side we are ready for them to accept.
			buffer := make([]byte, 1)
			if _, err := conn.Write(buffer); err != nil {
				d.logger.Printf("error sending please-accept message: %s\n",
					err)
				return nil, nil
			}
			return conn, nil
		}
	}
	return nil, nil
}

// Lookup a connection and remove it from the map. Caller must consume.
func (d *Dialer) lookup(address string) net.Conn {
	d.connectionMapLock.Lock()
	defer d.connectionMapLock.Unlock()
	if conn, ok := d.connectionMap[address]; ok {
		delete(d.connectionMap, address)
		return conn
	}
	return nil
}

func (d *Dialer) connectHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		d.logger.Println("not a hijacker ", req.RemoteAddr)
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		d.logger.Println("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	d.connectionMapLock.Lock()
	_, ok = d.connectionMap[req.RemoteAddr]
	d.connectionMapLock.Unlock()
	if ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusAlreadyReported)
		return
	}
	_, err = io.WriteString(conn, "HTTP/1.0 "+connectString+"\n\n")
	if err != nil {
		d.logger.Println("error writing connect message: ", err.Error())
		return
	}
	message := reverseDialerMessage{
		MinimumInterval: d.minimumInterval,
		MaximumInterval: d.maximumInterval,
	}
	encoder := json.NewEncoder(conn)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(message); err != nil {
		d.logger.Printf("error writing ReverseDialerMessage: %s\n", err)
		return
	}
	// Ensure we don't write anything else until the other end has drained it's
	// buffer.
	buffer := make([]byte, 1)
	if _, err := conn.Read(buffer); err != nil {
		d.logger.Printf("error reading sync byte: %s\n", err)
		return
	}
	if d.add(req.RemoteAddr, conn) {
		d.logger.Debugf(0, "Registered reverse dialer connection from: %s\n",
			req.RemoteAddr)
	} else {
		d.logger.Printf(
			"Closing duplicate reverse dialer connection from: %s\n",
			req.RemoteAddr)
	}
	conn = nil
}
