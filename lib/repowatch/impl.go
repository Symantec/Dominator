package repowatch

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

func checkDirectory(directory string) error {
	if fi, err := os.Stat(directory); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("not a directory: %s", directory)
	}
	return nil
}

func gitCommand(repositoryDirectory string, command ...string) error {
	cmd := exec.Command("git", command...)
	cmd.Dir = repositoryDirectory
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}
	return nil
}

func gitPull(repositoryDirectory string,
	latencyDistribution *tricorder.CumulativeDistribution) (string, error) {
	startTime := time.Now()
	if err := gitCommand(repositoryDirectory, "pull"); err != nil {
		return "", err
	}
	latencyDistribution.Add(time.Since(startTime))
	return readLatestCommitId(repositoryDirectory)
}

func readLatestCommitId(repositoryDirectory string) (string, error) {
	commitId, err := ioutil.ReadFile(
		filepath.Join(repositoryDirectory, ".git/refs/heads/master"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(commitId)), nil
}

func setupGitRepository(remoteURL, localDirectory string,
	latencyDistribution *tricorder.CumulativeDistribution) (string, error) {
	if err := os.MkdirAll(localDirectory, fsutil.DirPerms); err != nil {
		return "", err
	}
	gitSubdir := filepath.Join(localDirectory, ".git")
	if _, err := os.Stat(gitSubdir); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		err := gitCommand(localDirectory, "clone", remoteURL, ".")
		if err != nil {
			return "", err
		}
		return readLatestCommitId(localDirectory)
	} else {
		return gitPull(localDirectory, latencyDistribution)
	}
}

func watch(remoteURL, localDirectory string, checkInterval time.Duration,
	metricName string, logger log.DebugLogger) (<-chan string, error) {
	if checkInterval < time.Second {
		checkInterval = time.Second
	}
	if remoteURL == "" {
		return watchLocal(localDirectory, checkInterval, logger)
	}
	return watchGit(remoteURL, localDirectory, checkInterval, metricName,
		logger)
}

func watchGit(remoteURL, localDirectory string, checkInterval time.Duration,
	metricName string, logger log.DebugLogger) (<-chan string, error) {
	notificationChannel := make(chan string, 1)
	latencyDistribution := tricorder.NewGeometricBucketer(1, 1e5).
		NewCumulativeDistribution()
	err := tricorder.RegisterMetric(metricName, latencyDistribution,
		units.Millisecond, "latency of git pull calls")
	if err != nil {
		return nil, err
	}
	commitId, err := setupGitRepository(remoteURL, localDirectory,
		latencyDistribution)
	if err != nil {
		return nil, err
	}
	go watchGitLoop(localDirectory, commitId, checkInterval,
		latencyDistribution, notificationChannel, logger)
	return notificationChannel, nil
}

func watchGitLoop(directory, lastCommitId string, checkInterval time.Duration,
	latencyDist *tricorder.CumulativeDistribution,
	notificationChannel chan<- string, logger log.DebugLogger) {
	for ; ; time.Sleep(checkInterval) {
		if commitId, err := gitPull(directory, latencyDist); err != nil {
			logger.Println(err)
		} else if commitId != lastCommitId {
			lastCommitId = commitId
			notificationChannel <- directory
		}
	}
}

func watchLocal(directory string, checkInterval time.Duration,
	logger log.DebugLogger) (<-chan string, error) {
	if err := checkDirectory(directory); err != nil {
		return nil, err
	}
	notificationChannel := make(chan string, 1)
	go watchLocalLoop(directory, checkInterval, notificationChannel, logger)
	return notificationChannel, nil
}

func watchLocalLoop(directory string, checkInterval time.Duration,
	notificationChannel chan<- string, logger log.DebugLogger) {
	for ; ; time.Sleep(checkInterval) {
		if err := checkDirectory(directory); err != nil {
			logger.Println(err)
		} else {
			notificationChannel <- directory
		}
	}
}
