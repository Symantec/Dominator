package configurator

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Symantec/Dominator/lib/fsutil"
)

func (netconf *NetworkConfig) printDebian(writer io.Writer) error {
	fmt.Fprintln(writer,
		"# /etc/network/interfaces -- created by SmallStack installer")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "auto lo")
	fmt.Fprintln(writer, "iface lo inet loopback")
	for _, iface := range netconf.normalInterfaces {
		fmt.Fprintln(writer)
		name := iface.netInterface.Name
		if iface.subnet.Manage {
			name = fmt.Sprintf("br%d", iface.subnet.VlanId)
		}
		fmt.Fprintf(writer, "auto %s\n", name)
		fmt.Fprintf(writer, "iface %s inet static\n", name)
		fmt.Fprintf(writer, "\taddress      %s\n", iface.ipAddr)
		fmt.Fprintf(writer, "\tnetmask      %s\n", iface.subnet.IpMask)
		if iface.subnet.IpGateway.Equal(netconf.DefaultSubnet.IpGateway) {
			fmt.Fprintf(writer, "\tgateway      %s\n", iface.subnet.IpGateway)
		}
		if iface.subnet.Manage {
			fmt.Fprintf(writer, "\thwaddress    %s\n",
				iface.netInterface.HardwareAddr)
			fmt.Fprintf(writer, "\tbridge_ports %s\n", iface.netInterface.Name)
		}
	}
	for _, iface := range netconf.bridgeOnlyInterfaces {
		fmt.Fprintln(writer)
		name := fmt.Sprintf("br@%s", iface.subnetId)
		fmt.Fprintf(writer, "auto %s\n", name)
		fmt.Fprintf(writer, "iface %s inet manual\n", name)
		fmt.Fprintf(writer, "\thwaddress    %s\n",
			iface.netInterface.HardwareAddr)
		fmt.Fprintf(writer, "\tbridge_ports %s\n", iface.netInterface.Name)
	}
	if netconf.vlanRawDevice != "" {
		fmt.Fprintln(writer)
		fmt.Fprintf(writer, "auto %s\n", netconf.vlanRawDevice)
		fmt.Fprintf(writer, "iface %s inet manual\n", netconf.vlanRawDevice)
		if len(netconf.bondSlaves) > 1 {
			fmt.Fprintf(writer, "\tup ip link set %s mtu 9000\n",
				netconf.vlanRawDevice)
			fmt.Fprintln(writer, "\tbond-mode 802.3ad")
			fmt.Fprintln(writer, "\tbond-xmit_hash_policy 1")
			fmt.Fprint(writer, "\tslaves")
			for _, name := range netconf.bondSlaves {
				fmt.Fprint(writer, " ", name)
			}
			fmt.Fprintln(writer)
		}
		for _, iface := range netconf.bondedInterfaces {
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto %s\n", iface.name)
			fmt.Fprintf(writer, "iface %s inet static\n", iface.name)
			fmt.Fprintln(writer, "\tvlan-raw-device bond0")
			fmt.Fprintf(writer, "\taddress %s\n", iface.ipAddr)
			fmt.Fprintf(writer, "\tnetmask %s\n", iface.subnet.IpMask)
			if iface.subnet.IpGateway.Equal(netconf.DefaultSubnet.IpGateway) {
				fmt.Fprintf(writer, "\tgateway %s\n", iface.subnet.IpGateway)
			}
		}
		for _, vlanId := range netconf.bridges {
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto %s.%d\n", netconf.vlanRawDevice, vlanId)
			fmt.Fprintf(writer, "iface %s.%d inet manual\n",
				netconf.vlanRawDevice, vlanId)
			fmt.Fprintf(writer, "\tvlan-raw-device %s\n", netconf.vlanRawDevice)
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto br%d\n", vlanId)
			fmt.Fprintf(writer, "iface br%d inet manual\n", vlanId)
			fmt.Fprintf(writer, "\tbridge_ports %s.%d\n",
				netconf.vlanRawDevice, vlanId)
		}
	}
	return nil
}

func (netconf *NetworkConfig) updateDebian(rootDir string) (bool, error) {
	buffer := &bytes.Buffer{}
	if err := netconf.printDebian(buffer); err != nil {
		return false, err
	}
	filename := filepath.Join(rootDir, "etc", "network", "interfaces")
	// Check if it was written by me.
	if file, err := os.Open(filename); err != nil {
		return false, err
	} else {
		defer file.Close()
		fileBuffer := make([]byte, 256)
		_, err := io.ReadFull(file, fileBuffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return false, err
		}
		splitLines := strings.Split(string(fileBuffer), "\n")
		if len(splitLines) < 1 {
			return false, fmt.Errorf("%s is empty", filename)
		}
		if !strings.Contains(splitLines[0], "created by SmallStack") {
			return false, fmt.Errorf("%s not created by SmallStack", filename)
		}
	}
	if changed, err := fsutil.UpdateFile(buffer.Bytes(), filename); err != nil {
		return false, err
	} else if !changed {
		return false, nil
	}
	cmd := exec.Command("ifup", "-a")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
}

func (netconf *NetworkConfig) writeDebian(rootDir string) error {
	filename := filepath.Join(rootDir, "etc", "network", "interfaces")
	file, err := fsutil.CreateRenamingWriter(filename, fsutil.PublicFilePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	if err := netconf.printDebian(writer); err != nil {
		return err
	}
	return writer.Flush()
}
