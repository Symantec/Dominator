package main

import (
	"os"
	"path"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/verstr"
	"github.com/Symantec/Dominator/proto/sub"
)

func loadConfiguration(confDir string, conf *sub.Configuration,
	logger log.Logger) {
	file, err := os.Open(confDir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Println(err)
		}
		return
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		logger.Println(err)
		return
	}
	verstr.Sort(names)
	for _, name := range names {
		filename := path.Join(confDir, name)
		if err := json.ReadFromFile(filename, conf); err != nil {
			if !os.IsNotExist(err) {
				logger.Println(err)
			}
		}
	}
}
