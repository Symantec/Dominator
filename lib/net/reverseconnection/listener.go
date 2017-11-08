package reverseconnection

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	libnet "github.com/Symantec/Dominator/lib/net"
)

const (
	configDirectory = "/etc/reverse-listeners"
)

var (
	errorNotFound = errors.New("HTTP method not found")
)

func getIp4Address(conn net.Conn) (ip4Address, error) {
	remoteAddr := conn.RemoteAddr()
	var zero ip4Address
	if remoteAddr.Network() != "tcp" {
		return zero, errors.New("wrong network type: " + remoteAddr.Network())
	}
	remoteHost, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		return zero, err
	}
	return getIp4AddressFromAddress(remoteHost)
}

func getIp4AddressFromAddress(address string) (ip4Address, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return ip4Address{}, errors.New("failed to parse: " + address)
	}
	ip = ip.To4()
	if ip == nil {
		return ip4Address{}, errors.New(address + " is not IPv4")
	}
	return ip4Address{ip[0], ip[1], ip[2], ip[3]}, nil
}

func listen(network string, portNumber uint, logger log.DebugLogger) (
	*Listener, error) {
	rListener, err := libnet.ListenWithReuse(network,
		fmt.Sprintf(":%d", portNumber))
	if err != nil {
		return nil, err
	}
	acceptChannel := make(chan acceptEvent, 1)
	listener := &Listener{
		listener:      rListener,
		portNumber:    portNumber,
		logger:        logger,
		acceptChannel: acceptChannel,
		connectionMap: make(map[ip4Address]uint),
	}
	go listener.listen(acceptChannel)
	return listener, nil
}

func sleep(minInterval, maxInterval time.Duration) {
	jit := (maxInterval - minInterval) * time.Duration((rand.Intn(1000))) / 1000
	time.Sleep(minInterval + jit)
}

func (conn *Conn) close() error {
	if ip, err := getIp4Address(conn); err != nil {
		conn.listener.logger.Println(err)
	} else {
		conn.listener.forget(conn.RemoteAddr().String(), ip)
	}
	return conn.TCPConn.Close()
}

func (l *Listener) accept() (*Conn, error) {
	if l.closed {
		return nil, errors.New("listener is closed")
	}
	event := <-l.acceptChannel
	return event.conn, event.error
}

func (l *Listener) close() error {
	l.closed = true
	return l.listener.Close()
}

func (l *Listener) forget(remoteHost string, ip ip4Address) {
	l.logger.Debugf(1, "reverse listener: forget(%s)\n", remoteHost)
	l.connectionMapLock.Lock()
	defer l.connectionMapLock.Unlock()
	if numConn := l.connectionMap[ip]; numConn < 1 {
		panic("unknown connection from: " + remoteHost)
	} else {
		l.connectionMap[ip] = numConn - 1
	}
}

func (l *Listener) listen(acceptChannel chan<- acceptEvent) {
	for {
		if l.closed {
			break
		}
		conn, err := l.listener.Accept()
		tcpConn, ok := conn.(libnet.TCPConn)
		if !ok {
			conn.Close()
			l.logger.Println("rejecting non-TCP connection")
			continue
		}
		l.remember(conn)
		acceptChannel <- acceptEvent{&Conn{TCPConn: tcpConn, listener: l}, err}
	}
}

func (l *Listener) remember(conn net.Conn) {
	l.logger.Debugf(1, "reverse listener: remember(%s): %p\n",
		conn.RemoteAddr(), conn)
	if ip, err := getIp4Address(conn); err == nil {
		l.connectionMapLock.Lock()
		defer l.connectionMapLock.Unlock()
		l.connectionMap[ip]++
	}
}

func (l *Listener) requestConnections(serviceName string) error {
	var config ReverseListenerConfig
	filename := path.Join(configDirectory, serviceName)
	if err := libjson.ReadFromFile(filename, &config); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if config.Network == "" {
		config.Network = "tcp"
	}
	if config.MinimumInterval < time.Minute {
		config.MinimumInterval = time.Minute
	}
	if config.MaximumInterval <= config.MinimumInterval {
		config.MaximumInterval = config.MinimumInterval * 11 / 10
	}
	serverHost, _, err := net.SplitHostPort(config.ServerAddress)
	if err != nil {
		return err
	}
	go l.connectLoop(config, serverHost)
	return nil
}

func (l *Listener) connectLoop(config ReverseListenerConfig,
	serverHost string) {
	logger := prefixlogger.New("reverse listener: "+config.ServerAddress+": ",
		l.logger)
	logger.Debugf(0, "starting loop, min interval: %s, max interval: %s\n",
		config.MinimumInterval, config.MaximumInterval)
	for {
		sleep(config.MinimumInterval, config.MaximumInterval)
		addrs, err := net.LookupHost(serverHost)
		if err != nil {
			logger.Println(err)
			continue
		}
		foundExisting := false
		for _, addr := range addrs {
			if ip, err := getIp4AddressFromAddress(addr); err != nil {
				continue
			} else {
				l.connectionMapLock.Lock()
				if l.connectionMap[ip] > 0 {
					foundExisting = true
				}
				l.connectionMapLock.Unlock()
			}
			if foundExisting {
				break
			}
		}
		if foundExisting {
			continue
		}
		message, err := l.connect(config.Network, config.ServerAddress,
			config.MinimumInterval>>1, logger)
		if err != nil {
			if err != errorNotFound {
				logger.Println(err)
			}
			continue
		}
		if message.MinimumInterval >= time.Second {
			newMinimumInterval := message.MinimumInterval
			newMaximumInterval := config.MaximumInterval
			if message.MaximumInterval > newMinimumInterval {
				newMaximumInterval = message.MaximumInterval
			} else {
				newMaximumInterval = newMinimumInterval * 11 / 10
			}
			if newMinimumInterval != config.MinimumInterval ||
				newMaximumInterval != config.MaximumInterval {
				logger.Debugf(0,
					"min interval: %s -> %s, max interval: %s -> %s\n",
					config.MinimumInterval, newMinimumInterval,
					config.MaximumInterval, newMaximumInterval)
			}
			config.MinimumInterval = newMinimumInterval
			config.MaximumInterval = newMaximumInterval
		}
	}
}

func (l *Listener) connect(network, serverAddress string, timeout time.Duration,
	logger log.DebugLogger) (*reverseDialerMessage, error) {
	logger.Debugln(0, "dialing")
	localAddr := fmt.Sprintf(":%d", l.portNumber)
	deadline := time.Now().Add(timeout)
	rawConn, err := libnet.BindAndDial(network, localAddr, serverAddress,
		timeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rawConn != nil {
			rawConn.Close()
		}
	}()
	tcpConn, ok := rawConn.(libnet.TCPConn)
	if !ok {
		return nil, errors.New("rejecting non-TCP connection")
	}
	if err := rawConn.SetDeadline(deadline); err != nil {
		return nil, errors.New("error setting deadline: " + err.Error())
	}
	logger.Debugln(0, "sending HTTP CONNECT")
	_, err = io.WriteString(rawConn, "CONNECT "+urlPath+" HTTP/1.0\n\n")
	if err != nil {
		return nil, errors.New("error writing CONNECT: " + err.Error())
	}
	reader := bufio.NewReader(rawConn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
	if err != nil {
		return nil, errors.New("error reading HTTP response: " + err.Error())
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errorNotFound
	}
	if resp.StatusCode != http.StatusOK || resp.Status != connectString {
		return nil, errors.New("unexpected HTTP response: " + resp.Status)
	}
	decoder := json.NewDecoder(reader)
	var message reverseDialerMessage
	if err := decoder.Decode(&message); err != nil {
		return nil, errors.New("error decoding message: " + err.Error())
	}
	// Send all-clear to other side to ensure nothing further is buffered.
	buffer := make([]byte, 1)
	if _, err := rawConn.Write(buffer); err != nil {
		return nil, errors.New("error writing sync byte: " + err.Error())
	}
	if err := rawConn.SetDeadline(time.Time{}); err != nil {
		return nil, errors.New("error resetting deadline: " + err.Error())
	}
	logger.Println("made connection, waiting for remote consumption")
	// Wait for other side to consume.
	if _, err := rawConn.Read(buffer); err != nil {
		return nil, errors.New("error reading sync byte: " + err.Error())
		return nil, err
	}
	logger.Println("remote has consumed, injecting to local listener")
	l.remember(rawConn)
	l.acceptChannel <- acceptEvent{&Conn{TCPConn: tcpConn, listener: l}, nil}
	rawConn = nil // Prevent Close on return.
	return &message, nil
}
