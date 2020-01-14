package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	objclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
	"github.com/Cloud-Foundations/Dominator/lib/wsyscall"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func pushFileSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := pushFile(srpcClient, args[0], args[1]); err != nil {
		return fmt.Errorf("Error pushing file: %s", err)
	}
	return nil
}

func pushFile(srpcClient *srpc.Client, source, dest string) error {
	var sourceStat wsyscall.Stat_t
	if err := wsyscall.Stat(source, &sourceStat); err != nil {
		return err
	}
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	objClient := objclient.AttachObjectClient(srpcClient)
	defer objClient.Close()
	hashVal, _, err := objClient.AddObject(sourceFile, uint64(sourceStat.Size),
		nil)
	if err != nil {
		return err
	}
	newRegularInode := &filesystem.RegularInode{
		Mode:             filesystem.FileMode(sourceStat.Mode),
		Uid:              sourceStat.Uid,
		Gid:              sourceStat.Gid,
		MtimeNanoSeconds: int32(sourceStat.Mtim.Nsec),
		MtimeSeconds:     sourceStat.Mtim.Sec,
		Size:             uint64(sourceStat.Size),
		Hash:             hashVal}
	newInode := sub.Inode{Name: dest, GenericInode: newRegularInode}
	var updateRequest sub.UpdateRequest
	var updateReply sub.UpdateResponse
	updateRequest.Wait = true
	updateRequest.InodesToMake = append(updateRequest.InodesToMake, newInode)
	if *triggersFile != "" {
		updateRequest.Triggers, err = triggers.Load(*triggersFile)
		if err != nil {
			return err
		}
	} else if *triggersString != "" {
		updateRequest.Triggers, err = triggers.Decode([]byte(*triggersString))
		if err != nil {
			return err
		}
	}
	startTime := showStart("Subd.Update()")
	err = client.CallUpdate(srpcClient, updateRequest, &updateReply)
	showTimeTaken(startTime)
	return err
}
