package main

import (
	"flag"
	"fmt"
	"os"
	"path"
)

var (
	rootDir = flag.String("rootDir", "",
		"Name of root of directory tree to convert to an object cache")
)

func main() {
	flag.Parse()
	if *rootDir == "" {
		fmt.Fprintf(os.Stderr, "rootDir unspecified\n")
		os.Exit(1)
	}
	subdDirPathname := path.Join(*rootDir, ".subd")
	if err := os.RemoveAll(subdDirPathname); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	objectsDir := path.Join(subdDirPathname, "objects")
	if err := os.MkdirAll(objectsDir, 0750); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := walk(*rootDir, "/", objectsDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
