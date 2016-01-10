package main

import (
	"bufio"
	"errors"
	"fmt"
	imgclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"github.com/Symantec/Dominator/proto/sub"
	subclient "github.com/Symantec/Dominator/sub/client"
	"io/ioutil"
	"net/rpc"
	"os"
	"os/exec"
)

func diffImageVImageSubcommand(imageClient *rpc.Client,
	imageSClient *srpc.Client, objectClient *objectclient.ObjectClient,
	args []string) {
	commonDiffSubcommand(imageSClient, args[0],
		args[1], getImage, args[2], getImage)
}

func diffImageVSubSubcommand(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	commonDiffSubcommand(imageSClient, args[0],
		args[1], getImage, args[2], pollImage)
}

func diffSubVImageSubcommand(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	commonDiffSubcommand(imageSClient, args[0],
		args[1], pollImage, args[2], getImage)
}

func diffSubVSubSubcommand(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	commonDiffSubcommand(imageSClient, args[0],
		args[1], pollImage, args[2], pollImage)
}

func commonDiffSubcommand(client *srpc.Client, tool string,
	lName string, lGetFunc func(client *srpc.Client, name string) (
		*filesystem.FileSystem, error),
	rName string, rGetFunc func(client *srpc.Client, name string) (
		*filesystem.FileSystem, error)) {
	lfs, err := lGetFunc(client, lName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting left image\t%s\n", err)
		os.Exit(1)
	}
	rfs, err := rGetFunc(client, rName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting right image\t%s\n", err)
		os.Exit(1)
	}
	err = diffImages(tool, lfs, rfs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error diffing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getImage(client *srpc.Client, name string) (
	*filesystem.FileSystem, error) {
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	if err := imgclient.CallGetImage(client, request, &reply); err != nil {
		return nil, err
	}
	if reply.Image == nil {
		return nil, errors.New(name + ": not found")
	}
	reply.Image.FileSystem.RebuildInodePointers()
	return reply.Image.FileSystem, nil
}

func pollImage(client *srpc.Client, name string) (
	*filesystem.FileSystem, error) {
	clientName := fmt.Sprintf("%s:%d", name, constants.SubPortNumber)
	srpcClient, err := srpc.DialHTTP("tcp", clientName)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing %s", err))
	}
	defer srpcClient.Close()
	var request sub.PollRequest
	var reply sub.PollResponse
	if err = subclient.CallPoll(srpcClient, request, &reply); err != nil {
		return nil, err
	}
	if reply.FileSystem == nil {
		return nil, errors.New("no poll data")
	}
	reply.FileSystem.RebuildInodePointers()
	return reply.FileSystem, nil
}

func diffImages(tool string, lfs, rfs *filesystem.FileSystem) error {
	lname, err := writeImage(lfs)
	defer os.Remove(lname)
	if err != nil {
		return err
	}
	rname, err := writeImage(rfs)
	defer os.Remove(rname)
	if err != nil {
		return err
	}
	cmd := exec.Command(tool, lname, rname)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func writeImage(fs *filesystem.FileSystem) (string, error) {
	file, err := ioutil.TempFile("", "imagetool")
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	return file.Name(), fs.List(writer)
}
