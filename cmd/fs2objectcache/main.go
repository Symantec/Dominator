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

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr,
			"Usage: fs2objectcache [flags...]")
		fmt.Fprintln(os.Stderr, "Common flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr,
			"This tool will convert a file-system tree to an object cache.")
		fmt.Fprintln(os.Stderr,
			"It is a building block for a secure and fast re-imaging pipeline.")
		fmt.Fprintln(os.Stderr,
			"It should only be run from a trusted boot image.")
	}
}

func main() {
	flag.Parse()
	if *rootDir == "" {
		fmt.Fprintln(os.Stderr, "rootDir unspecified")
		os.Exit(1)
	}
	if *rootDir == "/" {
		fmt.Fprintln(os.Stderr, "Cannot convert root file-system")
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
