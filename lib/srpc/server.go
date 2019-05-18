package srpc

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/Dominator/lib/x509util"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

const (
	connectString   = "200 Connected to Go SRPC"
	rpcPath         = "/_goSRPC_/"      // GOB coder.
	tlsRpcPath      = "/_go_TLS_SRPC_/" // GOB coder.
	gobTlsRpcPath   = "/_go_TLS_SRPC_/GOB"
	jsonTlsRpcPath  = "/_go_TLS_SRPC_/JSON"
	listMethodsPath = rpcPath + "listMethods"

	methodTypeRaw = iota
	methodTypeCoder
	methodTypeRequestReply
)

type methodWrapper struct {
	methodType                    int
	public                        bool
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
	blockMethod func(methodName string,
		authInfo *AuthInformation) (func(), error)
	grantMethod func(serviceMethod string, authInfo *AuthInformation) bool
}

var (
	defaultGrantMethod = func(serviceMethod string,
		authInfo *AuthInformation) bool {
		return false
	}
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
var typeOfDecoder = reflect.TypeOf((*Decoder)(nil)).Elem()
var typeOfEncoder = reflect.TypeOf((*Encoder)(nil)).Elem()
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

func init() {
	http.HandleFunc(rpcPath, unsecuredHttpHandler)
	http.HandleFunc(tlsRpcPath, gobTlsHttpHandler)
	http.HandleFunc(gobTlsRpcPath, gobTlsHttpHandler)
	http.HandleFunc(jsonTlsRpcPath, jsonTlsHttpHandler)
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

func defaultMethodBlocker(methodName string,
	authInfo *AuthInformation) (func(), error) {
	return nil, nil
}

func defaultMethodGranter(serviceMethod string,
	authInfo *AuthInformation) bool {
	return defaultGrantMethod(serviceMethod, authInfo)
}

func registerName(name string, rcvr interface{},
	options ReceiverOptions) error {
	receiver := receiverType{methods: make(map[string]*methodWrapper)}
	typeOfReceiver := reflect.TypeOf(rcvr)
	valueOfReceiver := reflect.ValueOf(rcvr)
	receiverMetricsDir, err := serverMetricsDir.RegisterDirectory(name)
	if err != nil {
		return err
	}
	publicMethods := make(map[string]struct{}, len(options.PublicMethods))
	for _, methodName := range options.PublicMethods {
		publicMethods[methodName] = struct{}{}
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
		if _, ok := publicMethods[method.Name]; ok {
			mVal.public = true
		}
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
	if granter, ok := rcvr.(MethodGranter); ok {
		receiver.grantMethod = granter.GrantMethod
	} else {
		receiver.grantMethod = defaultMethodGranter
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
		return &methodWrapper{methodType: methodTypeRaw, fn: fn}
	}
	if methodType.NumIn() == 4 {
		if methodType.In(1) != typeOfConn {
			return nil
		}
		// Coder Method needs four ins: receiver, *Conn, Decoder, Encoder.
		if methodType.In(2) == typeOfDecoder &&
			methodType.In(3) == typeOfEncoder {
			return &methodWrapper{
				methodType: methodTypeCoder,
				fn:         fn,
			}
		}
		// RequestReply Method needs four ins: receiver, *Conn, request, *reply.
		if methodType.In(3).Kind() == reflect.Ptr {
			return &methodWrapper{
				methodType:   methodTypeRequestReply,
				fn:           fn,
				requestType:  methodType.In(2),
				responseType: methodType.In(3).Elem(),
			}
		}
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
	if m.methodType != methodTypeRequestReply {
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
	httpHandler(w, req, false, &gobCoder{})
}

func gobTlsHttpHandler(w http.ResponseWriter, req *http.Request) {
	httpHandler(w, req, true, &gobCoder{})
}

func jsonTlsHttpHandler(w http.ResponseWriter, req *http.Request) {
	httpHandler(w, req, true, &jsonCoder{})
}

func httpHandler(w http.ResponseWriter, req *http.Request, doTls bool,
	makeCoder coderMaker) {
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
	if tlsRequired && req.TLS != nil {
		if serverTlsConfig == nil ||
			!checkVerifiedChains(req.TLS.VerifiedChains,
				serverTlsConfig.ClientCAs) {
			serverMetricsMutex.Lock()
			numRejectedServerConnections++
			serverMetricsMutex.Unlock()
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("not a hijacker ", req.RemoteAddr)
		return
	}
	unsecuredConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	connToClose := unsecuredConn
	defer func() {
		connToClose.Close()
	}()
	if tcpConn, ok := unsecuredConn.(net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Println("error setting keepalive: ", err.Error())
			return
		}
		if err := tcpConn.SetKeepAlivePeriod(time.Minute * 5); err != nil {
			log.Println("error setting keepalive period: ", err.Error())
			return
		}
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotAcceptable)
		log.Println("non-TCP connection")
		return
	}
	_, err = io.WriteString(unsecuredConn, "HTTP/1.0 "+connectString+"\n\n")
	if err != nil {
		log.Println("error writing connect message: ", err.Error())
		return
	}
	myConn := &Conn{remoteAddr: req.RemoteAddr}
	if doTls {
		var tlsConn *tls.Conn
		if req.TLS == nil {
			tlsConn = tls.Server(unsecuredConn, serverTlsConfig)
			connToClose = tlsConn
			if err := tlsConn.Handshake(); err != nil {
				serverMetricsMutex.Lock()
				numRejectedServerConnections++
				serverMetricsMutex.Unlock()
				log.Println(err)
				return
			}
		} else {
			if tlsConn, ok = unsecuredConn.(*tls.Conn); !ok {
				log.Println("not really a TLS connection")
				return
			}
		}
		myConn.isEncrypted = true
		myConn.username, myConn.permittedMethods, myConn.groupList, err =
			getAuth(tlsConn.ConnectionState())
		if err != nil {
			log.Println(err)
			return
		}
		myConn.ReadWriter = bufio.NewReadWriter(bufio.NewReader(tlsConn),
			bufio.NewWriter(tlsConn))
	} else {
		myConn.ReadWriter = bufrw
	}
	serverMetricsMutex.Lock()
	numOpenServerConnections++
	serverMetricsMutex.Unlock()
	handleConnection(myConn, makeCoder)
	serverMetricsMutex.Lock()
	numOpenServerConnections--
	serverMetricsMutex.Unlock()
}

func checkVerifiedChains(verifiedChains [][]*x509.Certificate,
	certPool *x509.CertPool) bool {
	for _, vChain := range verifiedChains {
		vSubject := vChain[0].RawIssuer
		for _, cSubject := range certPool.Subjects() {
			if bytes.Compare(vSubject, cSubject) == 0 {
				return true
			}
		}
	}
	return false
}

func getAuth(state tls.ConnectionState) (string, map[string]struct{},
	map[string]struct{}, error) {
	var username string
	permittedMethods := make(map[string]struct{})
	trustCertMethods := false
	if fullAuthCaCertPool == nil ||
		checkVerifiedChains(state.VerifiedChains, fullAuthCaCertPool) {
		trustCertMethods = true
	}
	var groupList map[string]struct{}
	for _, certChain := range state.VerifiedChains {
		for _, cert := range certChain {
			var err error
			if username == "" {
				username, err = x509util.GetUsername(cert)
				if err != nil {
					return "", nil, nil, err
				}
			}
			if len(groupList) < 1 {
				groupList, err = x509util.GetGroupList(cert)
				if err != nil {
					return "", nil, nil, err
				}
			}
			if trustCertMethods {
				pms, err := x509util.GetPermittedMethods(cert)
				if err != nil {
					return "", nil, nil, err
				}
				for method := range pms {
					permittedMethods[method] = struct{}{}
				}
			}
		}
	}
	return username, permittedMethods, groupList, nil
}

func handleConnection(conn *Conn, makeCoder coderMaker) {
	defer conn.callReleaseNotifier()
	defer conn.Flush()
	for ; ; conn.Flush() {
		conn.callReleaseNotifier()
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
		if err := method.call(conn, makeCoder); err != nil {
			if err != ErrorCloseClient {
				log.Println(err)
			}
			return
		}
	}
}

func (conn *Conn) callReleaseNotifier() {
	if releaseNotifier := conn.releaseNotifier; releaseNotifier != nil {
		releaseNotifier()
	}
	conn.releaseNotifier = nil
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
	if conn.checkMethodAccess(serviceMethod) {
		conn.haveMethodAccess = true
	} else if receiver.grantMethod(serviceName, conn.GetAuthInformation()) {
		conn.haveMethodAccess = true
	} else {
		conn.haveMethodAccess = false
		if !method.public {
			method.numDeniedCalls++
			return nil, ErrorAccessToMethodDenied
		}
	}
	authInfo := conn.GetAuthInformation()
	if rn, err := receiver.blockMethod(methodName, authInfo); err != nil {
		return nil, err
	} else {
		conn.releaseNotifier = rn
	}
	return method, nil
}

// Returns true if the method is permitted, else false if denied.
func (conn *Conn) checkMethodAccess(serviceMethod string) bool {
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

func (m *methodWrapper) call(conn *Conn, makeCoder coderMaker) error {
	m.numPermittedCalls++
	startTime := time.Now()
	err := m._call(conn, makeCoder)
	timeTaken := time.Since(startTime)
	if err == nil {
		m.successfulCallsDistribution.Add(timeTaken)
	} else {
		m.failedCallsDistribution.Add(timeTaken)
	}
	return err
}

func (m *methodWrapper) _call(conn *Conn, makeCoder coderMaker) error {
	connValue := reflect.ValueOf(conn)
	conn.Decoder = makeCoder.MakeDecoder(conn)
	conn.Encoder = makeCoder.MakeEncoder(conn)
	switch m.methodType {
	case methodTypeRaw:
		returnValues := m.fn.Call([]reflect.Value{connValue})
		errInter := returnValues[0].Interface()
		if errInter != nil {
			return errInter.(error)
		}
		return nil
	case methodTypeCoder:
		returnValues := m.fn.Call([]reflect.Value{
			connValue,
			reflect.ValueOf(conn.Decoder),
			reflect.ValueOf(conn.Encoder),
		})
		errInter := returnValues[0].Interface()
		if errInter != nil {
			return errInter.(error)
		}
		return nil
	case methodTypeRequestReply:
		request := reflect.New(m.requestType)
		response := reflect.New(m.responseType)
		if err := conn.Decode(request.Interface()); err != nil {
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
		return conn.Encode(response.Interface())
	}
	return errors.New("unknown method type")
}
