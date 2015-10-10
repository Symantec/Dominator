package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/proto/imageserver"
	"github.com/Symantec/Dominator/proto/sub"
	"io/ioutil"
	"net/rpc"
	"os"
	"os/exec"
)

func diffImageVImageSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := diffImageVImage(imageClient, args[0], args[1], args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error diffing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func diffImageVImage(client *rpc.Client, tool, limage, rimage string) error {
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

func diffImageVSubSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := diffImageVSub(imageClient, args[0], args[1], args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error diffing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func diffImageVSub(client *rpc.Client, tool, image, sub string) error {
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

func getImage(client *rpc.Client, name string) (*filesystem.FileSystem, error) {
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	err := client.Call("ImageServer.GetImage", request, &reply)
	if err != nil {
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
	client, err := rpc.DialHTTP("tcp", clientName)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing %s", err))
	}
	var request sub.PollRequest
	var reply sub.PollResponse
	err = client.Call("Subd.Poll", request, &reply)
	if err != nil {
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
