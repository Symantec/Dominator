package slavedriver

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type slaveRoll struct {
	BusySlaves []SlaveInfo `json:",omitempty"`
	IdleSlaves []SlaveInfo `json:",omitempty"`
	Zombies    []SlaveInfo `json:",omitempty"`
}

func dialWithRetry(network, address string,
	timeout time.Duration) (*srpc.Client, error) {
	stopTime := time.Now().Add(timeout)
	for ; time.Until(stopTime) >= 0; time.Sleep(time.Second) {
		client, err := srpc.DialHTTP(network, address, time.Second*5)
		if err != nil {
			continue
		}
		if err := client.SetKeepAlivePeriod(time.Second * 30); err != nil {
			client.Close()
			return nil, err
		}
		return client, nil

	}
	return nil, fmt.Errorf("timed out connecting to: %s", address)
}

func listSlaves(slaves map[*Slave]struct{}) []*Slave {
	list := make([]*Slave, 0, len(slaves))
	for slave := range slaves {
		list = append(list, slave)
	}
	return list
}

func newSlaveDriver(options SlaveDriverOptions, slaveTrader SlaveTrader,
	logger log.DebugLogger) (*SlaveDriver, error) {
	if options.MinimumIdleSlaves < 1 {
		options.MinimumIdleSlaves = 1
	}
	if options.MaximumIdleSlaves < 1 {
		options.MaximumIdleSlaves = 1
	}
	if options.MaximumIdleSlaves < options.MinimumIdleSlaves {
		options.MaximumIdleSlaves = options.MinimumIdleSlaves
	}
	rollCallTrigger := make(chan struct{}, 1)
	driver := &SlaveDriver{
		options:         options,
		logger:          logger,
		rollCallTrigger: rollCallTrigger,
		slaveTrader:     slaveTrader,
		busySlaves:      make(map[*Slave]struct{}),
		idleSlaves:      make(map[*Slave]struct{}),
		zombies:         make(map[*Slave]struct{}),
	}
	if err := driver.loadSlaves(); err != nil {
		driver.slaveTrader.Close()
		return nil, err
	}
	go driver.watchRoll(rollCallTrigger)
	return driver, nil
}

func (driver *SlaveDriver) createSlave() (*Slave, error) {
	if slaveInfo, err := driver.slaveTrader.CreateSlave(); err != nil {
		return nil, err
	} else {
		slave := &Slave{
			clientAddress: fmt.Sprintf("%s:%d", slaveInfo.IpAddress,
				driver.options.PortNumber),
			info:   slaveInfo,
			driver: driver,
		}
		slave.client, err = dialWithRetry("tcp", slave.clientAddress,
			time.Minute)
		if err != nil {
			e := driver.slaveTrader.DestroySlave(slaveInfo.Identifier)
			if e != nil {
				driver.logger.Printf("error destroying: %s: %s\n",
					slaveInfo.IpAddress, e)
			}
			return nil, fmt.Errorf("error dialing: %s: %s",
				slave.clientAddress, err)
		}
		driver.logger.Printf("created slave: %s\n", slaveInfo.Identifier)
		return slave, nil
	}
}

func (driver *SlaveDriver) getIdleSlave() *Slave {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	for slave := range driver.idleSlaves {
		driver.busySlaves[slave] = struct{}{}
		delete(driver.idleSlaves, slave)
		driver.scheduleRollCall()
		return slave
	}
	return nil
}

func (driver *SlaveDriver) getSlave() (*Slave, error) {
	if slave := driver.getIdleSlave(); slave != nil {
		return slave, nil
	}
	driver.logger.Debugln(0, "creating slave")
	if slave, err := driver.createSlave(); err != nil {
		return nil, err
	} else {
		driver.mutex.Lock()
		driver.busySlaves[slave] = struct{}{}
		driver.mutex.Unlock()
		driver.scheduleRollCall()
		return slave, nil
	}
}

func (driver *SlaveDriver) loadSlaves() error {
	var slaves slaveRoll
	err := json.ReadFromFile(driver.options.DatabaseFilename, &slaves)
	if err != nil {
		if os.IsNotExist(err) {
			go driver.rollCall(false)
			return nil
		}
		return err
	}
	slaves.BusySlaves = append(slaves.BusySlaves, slaves.Zombies...)
	for _, slaveInfo := range slaves.BusySlaves {
		driver.zombies[&Slave{
			driver: driver,
			info:   slaveInfo,
		}] = struct{}{}
	}
	for _, slaveInfo := range slaves.IdleSlaves {
		slave := &Slave{
			clientAddress: fmt.Sprintf("%s:%d", slaveInfo.IpAddress,
				driver.options.PortNumber),
			info:   slaveInfo,
			driver: driver,
		}
		slave.client, err = dialWithRetry("tcp", slave.clientAddress,
			time.Minute)
		if err != nil {
			driver.logger.Printf("error dialing: %s: %s\n", slave.clientAddress,
				err)
			driver.zombies[slave] = struct{}{}
		} else {
			driver.idleSlaves[slave] = struct{}{}
		}
	}
	go driver.rollCall(false)
	return nil
}

// rollCall can take a while. It should be called from a goroutine.
func (driver *SlaveDriver) rollCall(writeState bool) {
	var numToCreate int
	driver.mutex.Lock()
	if uint(len(driver.idleSlaves)) > driver.options.MaximumIdleSlaves {
		for slave := range driver.idleSlaves {
			if uint(len(driver.idleSlaves)) <=
				driver.options.MaximumIdleSlaves {
				break
			}
			delete(driver.idleSlaves, slave)
			driver.zombies[slave] = struct{}{}
			writeState = true
		}
	} else {
		numToCreate = int(driver.options.MinimumIdleSlaves) -
			len(driver.idleSlaves)
	}
	zombies := listSlaves(driver.zombies)
	driver.mutex.Unlock()
	for _, slave := range zombies {
		err := driver.slaveTrader.DestroySlave(slave.info.Identifier)
		if err != nil {
			driver.logger.Printf("error destroying: %s: %s\n",
				slave.clientAddress, err)
		} else {
			driver.mutex.Lock()
			delete(driver.zombies, slave)
			driver.mutex.Unlock()
			writeState = true
		}
	}
	if numToCreate > 0 {
		driver.logger.Debugf(0, "creating %d slaves for idle pool\n",
			numToCreate)
		for i := 0; i < numToCreate; i++ {
			if slave, err := driver.createSlave(); err != nil {
				driver.logger.Println(err)
			} else {
				driver.mutex.Lock()
				driver.idleSlaves[slave] = struct{}{}
				driver.mutex.Unlock()
				writeState = true
			}
		}
	}
	if writeState {
		var slaves slaveRoll
		driver.mutex.Lock()
		for slave := range driver.busySlaves {
			slaves.BusySlaves = append(slaves.BusySlaves, slave.info)
		}
		for slave := range driver.idleSlaves {
			slaves.IdleSlaves = append(slaves.IdleSlaves, slave.info)
		}
		for slave := range driver.zombies {
			slaves.Zombies = append(slaves.Zombies, slave.info)
		}
		driver.mutex.Unlock()
		err := json.WriteToFile(driver.options.DatabaseFilename,
			fsutil.PublicFilePerms, "    ", slaves)
		if err != nil {
			driver.logger.Println(err)
		}
	}
}

func (driver *SlaveDriver) scheduleRollCall() {
	select {
	case driver.rollCallTrigger <- struct{}{}:
	default:
	}
}

func (driver *SlaveDriver) watchRoll(rollCallTrigger <-chan struct{}) {
	timer := time.NewTimer(time.Minute)
	for {
		select {
		case <-rollCallTrigger:
			timer.Reset(time.Minute)
			driver.rollCall(true)
		case <-timer.C:
			timer.Reset(time.Minute)
			driver.rollCall(false)
		}
	}
}

func (driver *SlaveDriver) writeHtml(writer io.Writer) {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	if len(driver.busySlaves) < 1 && len(driver.idleSlaves) < 1 &&
		len(driver.zombies) < 1 {
		fmt.Fprintf(writer, "No slaves for %s<br>\n", driver.options.Purpose)
		return
	}
	fmt.Fprintf(writer, "Slaves for %s:<br>\n", driver.options.Purpose)
	for slave := range driver.busySlaves {
		fmt.Fprintf(writer,
			"&nbsp;&nbsp;<a href=\"http://%s:%d/\">%s</a> (busy)<br>\n",
			slave, driver.options.PortNumber, slave)
	}
	for slave := range driver.idleSlaves {
		fmt.Fprintf(writer,
			"&nbsp;&nbsp;<a href=\"http://%s:%d/\">%s</a> (idle)<br>\n",
			slave, driver.options.PortNumber, slave)
	}
	for slave := range driver.zombies {
		fmt.Fprintf(writer,
			"&nbsp;&nbsp;<a href=\"http://%s:%d/\">%s</a> (zombie)<br>\n",
			slave, driver.options.PortNumber, slave)
	}
}

func (slave *Slave) destroy() {
	driver := slave.driver
	driver.logger.Printf("destroying slave: %s\n", slave.info.Identifier)
	driver.mutex.Lock()
	if _, ok := driver.busySlaves[slave]; !ok {
		driver.mutex.Unlock()
		panic("destroying slave which is not busy")
	}
	go slave.destroyAndUnlock()
}

func (slave *Slave) destroyAndUnlock() {
	driver := slave.driver
	defer driver.mutex.Unlock()
	if err := slave.client.Close(); err != nil {
		driver.logger.Println(err)
	}
	err := slave.driver.slaveTrader.DestroySlave(slave.info.Identifier)
	delete(driver.busySlaves, slave)
	if err != nil {
		driver.logger.Println(err)
		driver.zombies[slave] = struct{}{}
	} else {
	}
	driver.scheduleRollCall()
}

func (slave *Slave) getClient() *srpc.Client {
	return slave.client
}

func (slave *Slave) release() {
	driver := slave.driver
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	if _, ok := driver.idleSlaves[slave]; ok {
		panic("releasing idle slave")
	}
	if _, ok := driver.zombies[slave]; ok {
		panic("releasing zombie")
	}
	if _, ok := driver.busySlaves[slave]; !ok {
		panic("releasing unknown slave")
	}
	delete(driver.busySlaves, slave)
	driver.idleSlaves[slave] = struct{}{}
	driver.scheduleRollCall()
}
