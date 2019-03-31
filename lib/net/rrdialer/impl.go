package rrdialer

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
)

const weight = 0.2

var (
	pid = strconv.FormatInt(int64(os.Getpid()), 10)
)

type endpointType struct {
	address                    string // Host:port
	conn                       net.Conn
	dialing                    bool
	err                        error
	LastUpdate                 time.Time
	LatencyVariance            float64 // Seconds^2.
	MaximumLatency             float64 // Seconds.
	MeanLatency                float64 // Seconds.
	MinimumLatency             float64 // Seconds.
	standardDeviationOfLatency float64 // Seconds.
}

func getFastestEndpoint(endpoints []*endpointType) *endpointType {
	var fastestEndpoint *endpointType
	for _, endpoint := range endpoints {
		if endpoint.dialing {
			continue
		}
		if (fastestEndpoint == nil) ||
			(endpoint.MeanLatency > 0 &&
				endpoint.MeanLatency < fastestEndpoint.MeanLatency) {
			fastestEndpoint = endpoint
		}
	}
	return fastestEndpoint
}

func getHomeDirectory() (string, error) {
	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return homeDir, nil
	}
	if usr, err := user.Current(); err != nil {
		return "", err
	} else {
		return usr.HomeDir, nil
	}
}

func getMostStaleEndpoint(endpoints []*endpointType) *endpointType {
	var mostStaleEndpoint *endpointType
	for _, endpoint := range endpoints {
		if endpoint.dialing {
			continue
		}
		if (mostStaleEndpoint == nil) ||
			endpoint.LastUpdate.Before(mostStaleEndpoint.LastUpdate) {
			mostStaleEndpoint = endpoint
		}
	}
	return mostStaleEndpoint
}

func newDialer(dialer *net.Dialer, cacheDir string,
	logger log.DebugLogger) (*Dialer, error) {
	rrDialer := &Dialer{
		logger:    logger,
		rawDialer: dialer,
	}
	if cacheDir == "" {
		homedir, err := getHomeDirectory()
		if err != nil {
			return nil, err
		}
		cacheDir = filepath.Join(homedir, ".cache")
	}
	rrDialer.dirname = filepath.Join(cacheDir, "round-robin-dialer")
	return rrDialer, nil
}

func (d *Dialer) loadEndpointHistories(hostAddrs []string,
	port string) ([]*endpointType, error) {
	endpoints := make([]*endpointType, 0, len(hostAddrs))
	for _, hostAddr := range hostAddrs {
		address := hostAddr + ":" + port
		if endpoint, err := d.loadEndpointHistory(address); err != nil {
			return nil, err
		} else {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints, nil
}

func (d *Dialer) loadEndpointHistory(address string) (*endpointType, error) {
	filename := filepath.Join(d.dirname, address)
	var endpoint endpointType
	if err := json.ReadFromFile(filename, &endpoint); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return &endpointType{address: address}, nil
	} else {
		endpoint.address = address
		endpoint.computeStandardDeviationOfLatency()
		return &endpoint, nil
	}
}

func (d *Dialer) dialContext(ctx context.Context, network,
	address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	resolver := d.rawDialer.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	hostAddrs, err := resolver.LookupHost(context.Background(), host)
	if err != nil {
		return nil, err
	}
	if len(hostAddrs) < 1 {
		return nil, fmt.Errorf("no addresses found for: %s", host)
	} else if len(hostAddrs) == 1 {
		return d.rawDialer.DialContext(ctx, network, hostAddrs[0]+":"+port)
	}
	logLevel := int16(-1)
	if getter, ok := d.logger.(log.DebugLogLevelGetter); ok {
		logLevel = getter.GetLevel()
	}
	endpoints, err := d.loadEndpointHistories(hostAddrs, port)
	if err != nil {
		return nil, err
	}
	return d.dialEndpoints(ctx, network, address, endpoints, logLevel)
}

func (d *Dialer) dialEndpoints(ctx context.Context, network, address string,
	endpoints []*endpointType, logLevel int16) (net.Conn, error) {
	timeoutTimer := time.NewTimer(d.rawDialer.Timeout)
	results := make(chan *endpointType, len(endpoints))
	// Immediately dial the historically fastest endpoint.
	fastestEndpoint := getFastestEndpoint(endpoints)
	d.goDialEndpoint(ctx, network, fastestEndpoint, "fastest", results)
	impatienceTimerFastest := fastestEndpoint.makeImpatienceTimer()
	stalestEndpoint := getMostStaleEndpoint(endpoints)
	d.goDialEndpoint(ctx, network, stalestEndpoint, "oldest", results)
	impatienceTimerStalest := stalestEndpoint.makeImpatienceTimer()
	// Dial all endpoints without history or if debug mode is enabled.
	for _, endpoint := range endpoints {
		if logLevel >= 3 || endpoint.MeanLatency <= 0 {
			d.goDialEndpoint(ctx, network, endpoint, "all", results)
		}
	}
	failureCounter := 0
	problemCounter := 0
	for {
		select {
		case endpoint := <-results:
			if endpoint.err != nil {
				failureCounter++
				problemCounter++
				if failureCounter >= len(endpoints) {
					for _, endpoint := range endpoints {
						d.logger.Printf("error dialing: %s: %s\n",
							endpoint.address, endpoint.err)
					}
					return nil, fmt.Errorf("failed connecting to: %s", address)
				}
				for _, endpoint := range endpoints {
					d.goDialEndpoint(ctx, network, endpoint, "backups",
						results)
				}
				if problemCounter == 2 {
					d.logger.Println(
						"At least 2 endpoints have issues, dialed remaining endpoints")
				}
				break
			}
			d.logger.Debugf(2, "connected: %s\n", endpoint.conn.RemoteAddr())
			return endpoint.conn, nil
		case <-impatienceTimerFastest.C:
			problemCounter++
			for _, endpoint := range endpoints {
				d.goDialEndpoint(ctx, network, endpoint, "impatiently", results)
			}
			if problemCounter == 2 {
				d.logger.Println(
					"At least 2 endpoints have issues, dialed remaining endpoints")
			}
		case <-impatienceTimerStalest.C:
			problemCounter++
			for _, endpoint := range endpoints {
				d.goDialEndpoint(ctx, network, endpoint, "impatiently", results)
			}
			if problemCounter == 2 {
				d.logger.Println(
					"At least 2 endpoints have issues, dialed remaining endpoints")
			}
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timed out connecting to: %s", address)
		}
	}
}

func (d *Dialer) goDialEndpoint(ctx context.Context, network string,
	endpoint *endpointType, reason string, result chan<- *endpointType) {
	if endpoint.dialing {
		return
	}
	endpoint.dialing = true
	endpoint.LastUpdate = time.Now()
	d.logger.Debugf(2, "dialing %s: %s\n", reason, endpoint.address)
	d.waitGroup.Add(1)
	go func() {
		defer d.waitGroup.Done()
		startTime := time.Now()
		conn, err := d.rawDialer.DialContext(ctx, network, endpoint.address)
		if err != nil {
			endpoint.err = err
		} else {
			endpoint.conn = conn
			d.recordEvent(endpoint, time.Since(startTime).Seconds())
		}
		result <- endpoint
	}()
}

func (d *Dialer) recordEvent(endpoint *endpointType, latency float64) {
	if d.dirname == "" { // When testing.
		return
	}
	filename := filepath.Join(d.dirname, endpoint.address)
	tmpFilename := filepath.Join(d.dirname, endpoint.address+pid)
	endpoint.LastUpdate = time.Now()
	if endpoint.MeanLatency <= 0 {
		endpoint.MeanLatency = latency
	} else {
		delta := latency - endpoint.MeanLatency
		endpoint.MeanLatency = latency*weight +
			(1.0-weight)*endpoint.MeanLatency
		endpoint.LatencyVariance = (1.0 - weight) *
			(endpoint.LatencyVariance + weight*delta*delta)
	}
	endpoint.computeStandardDeviationOfLatency()
	d.logger.Debugf(3, "%s: L: %f ms, Lm: %f ms, Lsd: %f ms\n",
		endpoint.address, latency*1e3, endpoint.MeanLatency*1e3,
		endpoint.standardDeviationOfLatency*1e3)
	// endpoint.LastLatency = latency
	if latency > endpoint.MaximumLatency {
		endpoint.MaximumLatency = latency
	}
	if endpoint.MinimumLatency <= 0 || latency < endpoint.MinimumLatency {
		endpoint.MinimumLatency = latency
	}
	file, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		fsutil.PublicFilePerms)
	if err != nil {
		if os.IsNotExist(err) {
			if e := os.MkdirAll(d.dirname, fsutil.DirPerms); e != nil {
				d.logger.Println(err)
				d.logger.Println(e)
				return
			}
		}
		file, err = os.OpenFile(tmpFilename, os.O_CREATE|os.O_EXCL|os.O_WRONLY,
			fsutil.PublicFilePerms)
	}
	if err != nil {
		d.logger.Println(err)
		return
	}
	defer file.Close()
	defer os.Remove(tmpFilename)
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	if err := json.WriteWithIndent(writer, "    ", endpoint); err != nil {
		d.logger.Println(err)
		return
	}
	if err := writer.Flush(); err != nil {
		d.logger.Println(err)
		return
	}
	if err := file.Close(); err != nil {
		d.logger.Println(err)
		return
	}
	if err := os.Rename(tmpFilename, filename); err != nil {
		d.logger.Println(err)
		return
	}
}

func (d *Dialer) waitForBackgroundResults(timeout time.Duration) {
	finished := make(chan struct{}, 1)
	timer := time.NewTimer(timeout)
	go func(finished chan<- struct{}) {
		d.waitGroup.Wait()
		finished <- struct{}{}
	}(finished)
	select {
	case <-finished:
		timer.Stop()
	case <-timer.C:
	}
}

func (e *endpointType) computeStandardDeviationOfLatency() {
	if e.LatencyVariance <= 0 {
		return
	}
	e.standardDeviationOfLatency = math.Sqrt(e.LatencyVariance)
}

func (e *endpointType) makeImpatienceTimer() *time.Timer {
	if e.LatencyVariance <= 0 {
		timer := time.NewTimer(time.Second)
		timer.Stop()
		return timer
	}
	timeoutDelta := e.MeanLatency * 0.1
	if td := 2 * e.standardDeviationOfLatency; td > timeoutDelta {
		timeoutDelta = td
	}
	return time.NewTimer(time.Duration(float64(time.Second) *
		(e.MeanLatency + timeoutDelta)))
}
