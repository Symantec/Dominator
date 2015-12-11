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
	err := diffImageVImage(imageSClient, args[0], args[1], args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error diffing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func diffImageVImage(client *srpc.Client, tool, limage, rimage string) error {
	lfs, err := getImage(client, limage)
	if err != nil {
		return err
	}
	rfs, err := getImage(client, rimage)
	if err != nil {
		return err
	}
	return diffImages(tool, lfs, rfs)
}

func diffImageVSubSubcommand(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := diffImageVSub(imageSClient, args[0], args[1], args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error diffing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func diffImageVSub(client *srpc.Client, tool, image, sub string) error {
	lfs, err := getImage(client, image)
	if err != nil {
		return err
	}
	rfs, err := pollImage(sub)
	if err != nil {
		return err
	}
	return diffImages(tool, lfs, rfs)
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

func pollImage(name string) (*filesystem.FileSystem, error) {
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
	reply.FileSystem.FileSystem.RebuildInodePointers()
	return &reply.FileSystem.FileSystem, nil
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
