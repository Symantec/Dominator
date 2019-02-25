package configurator

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/fsutil"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func printResolvConf(writer io.Writer, subnet *hyper_proto.Subnet) error {
	if subnet.DomainName != "" {
		fmt.Fprintf(writer, "domain %s\n", subnet.DomainName)
		fmt.Fprintf(writer, "search %s\n", subnet.DomainName)
		fmt.Fprintln(writer)
	}
	for _, nameserver := range subnet.DomainNameServers {
		fmt.Fprintf(writer, "nameserver %s\n", nameserver)
	}
	return nil
}

func updateResolvConf(rootDir string,
	subnet *hyper_proto.Subnet) (bool, error) {
	buffer := &bytes.Buffer{}
	if err := printResolvConf(buffer, subnet); err != nil {
		return false, err
	}
	filename := filepath.Join(rootDir, "etc", "resolv.conf")
	return fsutil.UpdateFile(buffer.Bytes(), filename)
}

func writeResolvConf(rootDir string, subnet *hyper_proto.Subnet) error {
	filename := filepath.Join(rootDir, "etc", "resolv.conf")
	file, err := fsutil.CreateRenamingWriter(filename, fsutil.PublicFilePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	if err := printResolvConf(writer, subnet); err != nil {
		return err
	}
	return writer.Flush()
}
