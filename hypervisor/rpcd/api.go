package rpcd

import (
	"io"
	"net"

	"github.com/Symantec/Dominator/hypervisor/manager"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type DhcpServer interface {
	AddNetbootLease(address proto.Address, hostname string,
		subnet *proto.Subnet) error
	MakeAcknowledgmentChannel(ipAddr net.IP) <-chan struct{}
	RemoveLease(ipAddr net.IP)
}

type srpcType struct {
	dhcpServer     DhcpServer
	logger         log.DebugLogger
	manager        *manager.Manager
	tftpbootServer TftpbootServer
}

type TftpbootServer interface {
	RegisterFiles(ipAddr net.IP, files map[string][]byte)
	UnregisterFiles(ipAddr net.IP)
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(manager *manager.Manager, dhcpServer DhcpServer,
	tftpbootServer TftpbootServer, logger log.DebugLogger) (
	*htmlWriter, error) {
	srpcObj := &srpcType{
		dhcpServer:     dhcpServer,
		logger:         logger,
		manager:        manager,
		tftpbootServer: tftpbootServer,
	}
	srpc.SetDefaultGrantMethod(
		func(_ string, authInfo *srpc.AuthInformation) bool {
			return manager.CheckOwnership(authInfo)
		})
	srpc.RegisterNameWithOptions("Hypervisor", srpcObj, srpc.ReceiverOptions{
		PublicMethods: []string{
			"AcknowledgeVm",
			"BecomePrimaryVmOwner",
			"ChangeVmOwnerUsers",
			"ChangeVmTags",
			"CommitImportedVm",
			"CreateVm",
			"DeleteVmVolume",
			"DestroyVm",
			"DiscardVmAccessToken",
			"DiscardVmOldImage",
			"DiscardVmOldUserData",
			"DiscardVmSnapshot",
			"GetUpdates",
			"GetVmAccessToken",
			"GetVmInfo",
			"GetVmUserData",
			"GetVmVolume",
			"ImportLocalVm",
			"ListVMs",
			"ListVolumeDirectories",
			"MigrateVm",
			"ProbeVmPort",
			"ReplaceVmImage",
			"ReplaceVmUserData",
			"RestoreVmFromSnapshot",
			"RestoreVmImage",
			"RestoreVmUserData",
			"SnapshotVm",
			"StartVm",
			"StopVm",
			"TraceVmMetadata",
		}})
	return (*htmlWriter)(srpcObj), nil
}
