package srpc

import (
	"bufio"
	"crypto/tls"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/x509util"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

const (
	connectString   = "200 Connected to Go SRPC"
	rpcPath         = "/_goSRPC_/"
	tlsRpcPath      = "/_go_TLS_SRPC_/"
	listMethodsPath = rpcPath + "listMethods"
)

type methodWrapper struct {
	plain                         bool
	fn                            reflect.Value
	requestType                   reflect.Type
	responseType                  reflect.Type
	failedCallsDistribution       *tricorder.CumulativeDistribution
	failedRRCallsDistribution     *tricorder.CumulativeDistribution
	numDeniedCalls                uint64
	numPermittedCalls             uint64
	successfulCallsDistribution   *tricorder.CumulativeDistribution
	successfulRRCallsDistribution *tricorder.CumulativeDistribution
}

type receiverType struct {
	methods     map[string]*methodWrapper
	blockMethod func(methodName string) bool
}

var (
	receivers                    map[string]receiverType = make(map[string]receiverType)
	serverMetricsDir             *tricorder.DirectorySpec
	bucketer                     *tricorder.Bucketer
	serverMetricsMutex           sync.Mutex
	numServerConnections         uint64
	numOpenServerConnections     uint64
	numRejectedServerConnections uint64
)

// Precompute some reflect types. Can't use the types directly because Typeof
// takes an empty interface value. This is annoying.
var typeOfConn = reflect.TypeOf((**Conn)(nil)).Elem()
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

func init() {
	http.HandleFunc(rpcPath, unsecuredHttpHandler)
	http.HandleFunc(tlsRpcPath, tlsHttpHandler)
	http.HandleFunc(listMethodsPath, listMethodsHttpHandler)
	registerServerMetrics()
}

func registerServerMetrics() {
	var err error
	serverMetricsDir, err = tricorder.RegisterDirectory("srpc/server")
	if err != nil {
		panic(err)
	}
	err = serverMetricsDir.RegisterMetric("num-connections",
		&numServerConnections, units.None, "number of connection attempts")
	if err != nil {
		panic(err)
	}
	err = serverMetricsDir.RegisterMetric("num-open-connections",
		&numOpenServerConnections, units.None, "number of open connections")
	if err != nil {
		panic(err)
	}
	err = serverMetricsDir.RegisterMetric("num-rejected-connections",
		&numRejectedServerConnections, units.None,
		"number of rejected connections")
	if err != nil {
		panic(err)
	}
	bucketer = tricorder.NewGeometricBucketer(0.1, 1e5)
}

func defaultMethodBlocker(methodName string) bool {
	return false
}

func registerName(name string, rcvr interface{}) error {
	var receiver receiverType
	receiver.methods = make(map[string]*methodWrapper)
	typeOfReceiver := reflect.TypeOf(rcvr)
	valueOfReceiver := reflect.ValueOf(rcvr)
	receiverMetricsDir, err := serverMetricsDir.RegisterDirectory(name)
	if err != nil {
		return err
	}
	for index := 0; index < typeOfReceiver.NumMethod(); index++ {
		method := typeOfReceiver.Method(index)
		if method.PkgPath != "" { // Method must be exported.
			continue
		}
		methodType := method.Type
		mVal := getMethod(methodType, valueOfReceiver.Method(index))
		if mVal == nil {
			continue
		}
		receiver.methods[method.Name] = mVal
		dir, err := receiverMetricsDir.RegisterDirectory(method.Name)
		if err != nil {
			return err
		}
		if err := mVal.registerMetrics(dir); err != nil {
			return err
		}
	}
	if blocker, ok := rcvr.(MethodBlocker); ok {
		receiver.blockMethod = blocker.BlockMethod
	} else {
		receiver.blockMethod = defaultMethodBlocker
	}
	receivers[name] = receiver
	return nil
}

func getMethod(methodType reflect.Type, fn reflect.Value) *methodWrapper {
	if methodType.NumOut() != 1 {
		return nil
	}
	if methodType.Out(0) != typeOfError {
		return nil
	}
	if methodType.NumIn() == 2 {
		// Method needs two ins: receiver, *Conn.
		if methodType.In(1) != typeOfConn {
			return nil
		}
		return &methodWrapper{plain: true, fn: fn}
	}
	if methodType.NumIn() == 4 {
		// Method needs four ins: receiver, *Conn, request, *reply.
		if methodType.In(1) != typeOfConn {
			return nil
		}
		if methodType.In(3).Kind() != reflect.Ptr {
			return nil
		}
		return &methodWrapper{
			plain:        false,
			fn:           fn,
			requestType:  methodType.In(2),
			responseType: methodType.In(3).Elem()}
	}
	return nil
}

func (m *methodWrapper) registerMetrics(dir *tricorder.DirectorySpec) error {
	m.failedCallsDistribution = bucketer.NewCumulativeDistribution()
	err := dir.RegisterMetric("failed-call-durations",
		m.failedCallsDistribution, units.Millisecond,
		"duration of failed calls")
	if err != nil {
		return err
	}
	err = dir.RegisterMetric("num-denied-calls", &m.numDeniedCalls,
		units.None, "number of denied calls to method")
	if err != nil {
		return err
	}
	err = dir.RegisterMetric("num-permitted-calls", &m.numPermittedCalls,
		units.None, "number of permitted calls to method")
	if err != nil {
		return err
	}
	m.successfulCallsDistribution = bucketer.NewCumulativeDistribution()
	err = dir.RegisterMetric("successful-call-durations",
		m.successfulCallsDistribution, units.Millisecond,
		"duration of successful calls")
	if err != nil {
		return err
	}
	if m.plain {
		return nil
	}
	m.failedRRCallsDistribution = bucketer.NewCumulativeDistribution()
	err = dir.RegisterMetric("failed-request-reply-call-durations",
		m.failedRRCallsDistribution, units.Millisecond,
		"duration of failed request-reply calls")
	if err != nil {
		return err
	}
	m.successfulRRCallsDistribution = bucketer.NewCumulativeDistribution()
	err = dir.RegisterMetric("successful-request-reply-call-durations",
		m.successfulRRCallsDistribution, units.Millisecond,
		"duration of successful request-reply calls")
	if err != nil {
		return err
	}
	return nil
}

func unsecuredHttpHandler(w http.ResponseWriter, req *http.Request) {
	httpHandler(w, req, false)
}

func tlsHttpHandler(w http.ResponseWriter, req *http.Request) {
	httpHandler(w, req, true)
}

func httpHandler(w http.ResponseWriter, req *http.Request, doTls bool) {
	serverMetricsMutex.Lock()
	numServerConnections++
	serverMetricsMutex.Unlock()
	if doTls && serverTlsConfig == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if (tlsRequired && !doTls) || req.Method != "CONNECT" {
		serverMetricsMutex.Lock()
		numRejectedServerConnections++
		serverMetricsMutex.Unlock()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	unsecuredConn, bufrw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Println("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	if tcpConn, ok := unsecuredConn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Println("error setting keepalive: ", err.Error())
			return
		}
		if err := tcpConn.SetKeepAlivePeriod(time.Minute * 5); err != nil {
			log.Println("error setting keepalive period: ", err.Error())
			return
		}
	}
	_, err = io.WriteString(unsecuredConn, "HTTP/1.0 "+connectString+"\n\n")
	if err != nil {
		log.Println("error writing connect message: ", err.Error())
		return
	}
	myConn := &Conn{remoteAddr: req.RemoteAddr}
	if doTls {
		tlsConn := tls.Server(unsecuredConn, serverTlsConfig)
		defer tlsConn.Close()
		if err := tlsConn.Handshake(); err != nil {
			serverMetricsMutex.Lock()
			numRejectedServerConnections++
			serverMetricsMutex.Unlock()
			log.Println(err)
			return
		}
		myConn.isEncrypted = true
		myConn.username, myConn.permittedMethods, err = getAuth(
			tlsConn.ConnectionState())
		if err != nil {
			log.Println(err)
			return
		}
		myConn.ReadWriter = bufio.NewReadWriter(bufio.NewReader(tlsConn),
			bufio.NewWriter(tlsConn))
	} else {
		defer unsecuredConn.Close()
		myConn.ReadWriter = bufrw
	}
	serverMetricsMutex.Lock()
	numOpenServerConnections++
	serverMetricsMutex.Unlock()
	handleConnection(myConn)
	serverMetricsMutex.Lock()
	numOpenServerConnections--
	serverMetricsMutex.Unlock()
}

func getAuth(state tls.ConnectionState) (string, map[string]struct{}, error) {
	var username string
	permittedMethods := make(map[string]struct{})
	for _, certChain := range state.VerifiedChains {
		for _, cert := range certChain {
			var err error
			if username == "" {
				username, err = x509util.GetUsername(cert)
				if err != nil {
					return "", nil, err
				}
			}
			pms, err := x509util.GetPermittedMethods(cert)
			if err != nil {
				return "", nil, err
			}
			for method := range pms {
				permittedMethods[method] = struct{}{}
			}
		}
	}
	return username, permittedMethods, nil
}

func handleConnection(conn *Conn) {
	defer conn.Flush()
	for ; ; conn.Flush() {
		serviceMethod, err := conn.ReadString('\n')
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			log.Println(err)
			if _, err := conn.WriteString(err.Error() + "\n"); err != nil {
				log.Println(err)
				return
			}
			continue
		}
		serviceMethod = serviceMethod[:len(serviceMethod)-1]
		if serviceMethod == "" {
			// Received a "ping" request, send response.
			if _, err := conn.WriteString("\n"); err != nil {
				log.Println(err)
				return
			}
			continue
		}
		method, err := conn.findMethod(serviceMethod)
		if err != nil {
			if _, err := conn.WriteString(err.Error() + "\n"); err != nil {
				log.Println(err)
				return
			}
			continue
		}
		// Method is OK to call. Tell client and then call method handler.
		if _, err := conn.WriteString("\n"); err != nil {
			log.Println(err)
			return
		}
		if err := conn.Flush(); err != nil {
			log.Println(err)
			return
		}
		if err := method.call(conn); err != nil {
			log.Println(err)
			return
		}
	}
}

func (conn *Conn) findMethod(serviceMethod string) (*methodWrapper, error) {
	splitServiceMethod := strings.Split(serviceMethod, ".")
	if len(splitServiceMethod) != 2 {
		return nil, errors.New("malformed Service.Method: " + serviceMethod)
	}
	serviceName := splitServiceMethod[0]
	receiver, ok := receivers[serviceName]
	if !ok {
		return nil, errors.New("unknown service: " + serviceName)
	}
	methodName := splitServiceMethod[1]
	method, ok := receiver.methods[methodName]
	if !ok {
		return nil, errors.New(serviceName + ": unknown method: " + methodName)
	}
	if !conn.checkPermitted(serviceMethod) {
		method.numDeniedCalls++
		return nil, ErrorAccessToMethodDenied
	}
	if receiver.blockMethod(methodName) {
		return nil, ErrorMethodBlocked
	}
	return method, nil
}

// Returns true if the method is permitted, else false if denied.
func (conn *Conn) checkPermitted(serviceMethod string) bool {
	if conn.permittedMethods == nil {
		return true
	}
	for sm := range conn.permittedMethods {
		if matched, _ := filepath.Match(sm, serviceMethod); matched {
			return true
		}
	}
	return false
}

func listMethodsHttpHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	methods := make([]string, len(receivers))
	for receiverName, receiver := range receivers {
		for method := range receiver.methods {
			methods = append(methods, receiverName+"."+method+"\n")
		}
	}
	sort.Strings(methods)
	for _, method := range methods {
		writer.WriteString(method)
	}
}

func (m *methodWrapper) call(conn *Conn) error {
	m.numPermittedCalls++
	startTime := time.Now()
	err := m._call(conn)
	timeTaken := time.Since(startTime)
	if err == nil {
		m.successfulCallsDistribution.Add(timeTaken)
	} else {
		m.failedCallsDistribution.Add(timeTaken)
	}
	return err
}

func (m *methodWrapper) _call(conn *Conn) error {
	connValue := reflect.ValueOf(conn)
	if m.plain {
		returnValues := m.fn.Call([]reflect.Value{connValue})
		errInter := returnValues[0].Interface()
		if errInter != nil {
			return errInter.(error)
		}
		return nil
	}
	request := reflect.New(m.requestType)
	response := reflect.New(m.responseType)
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(request.Interface()); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	startTime := time.Now()
	returnValues := m.fn.Call([]reflect.Value{connValue, request.Elem(),
		response})
	timeTaken := time.Since(startTime)
	errInter := returnValues[0].Interface()
	if errInter != nil {
		m.failedRRCallsDistribution.Add(timeTaken)
		err := errInter.(error)
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	m.successfulRRCallsDistribution.Add(timeTaken)
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response.Interface())
}
