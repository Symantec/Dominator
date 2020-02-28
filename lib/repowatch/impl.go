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

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"github.com/Cloud-Foundations/tricorder/go/tricorder/units"
)

type gitMetricsType struct {
	lastAttemptedPullTime  time.Time
	lastCommitId           string
	lastSuccessfulPullTime time.Time
	lastNotificationTime   time.Time
	latencyDistribution    *tricorder.CumulativeDistribution
}

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
	metrics *gitMetricsType) (string, error) {
	metrics.lastAttemptedPullTime = time.Now()
	if err := gitCommand(repositoryDirectory, "pull"); err != nil {
		return "", err
	}
	metrics.lastSuccessfulPullTime = time.Now()
	metrics.latencyDistribution.Add(
		metrics.lastSuccessfulPullTime.Sub(metrics.lastAttemptedPullTime))
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

func setupGitRepository(remoteURL, localDirectory, awsSecretId string,
	metrics *gitMetricsType, logger log.DebugLogger) (string, error) {
	if err := os.MkdirAll(localDirectory, fsutil.DirPerms); err != nil {
		return "", err
	}
	gitSubdir := filepath.Join(localDirectory, ".git")
	if _, err := os.Stat(gitSubdir); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		metrics.lastAttemptedPullTime = time.Now()
		if err := awsGetKey(awsSecretId, logger); err != nil {
			return "", err
		}
		err := gitCommand(localDirectory, "clone", remoteURL, ".")
		if err != nil {
			return "", err
		}
		metrics.lastSuccessfulPullTime = time.Now()
		return readLatestCommitId(localDirectory)
	} else {
		go func() {
			for {
				if err := awsGetKey(awsSecretId, logger); err != nil {
					logger.Println(err)
					time.Sleep(time.Minute * 5)
				} else {
					return
				}
			}
		}()
		// Try to be as fresh as possible.
		if commitId, err := gitPull(localDirectory, metrics); err != nil {
			logger.Println(err)
			return readLatestCommitId(localDirectory)
		} else {
			return commitId, nil
		}
	}
}

func watch(config Config, metricDirectory string,
	logger log.DebugLogger) (<-chan string, error) {
	if config.Branch != "" && config.Branch != "master" {
		return nil, errors.New("non-master branch not supported")
	}
	if config.CheckInterval < time.Second {
		config.CheckInterval = time.Second
	}
	if config.RepositoryURL == "" {
		return watchLocal(config.LocalRepositoryDirectory, config.CheckInterval,
			metricDirectory, logger)
	}
	return watchGit(config.RepositoryURL, config.LocalRepositoryDirectory,
		config.AwsSecretId, config.CheckInterval, metricDirectory, logger)
}

func watchGit(remoteURL, localDirectory, awsSecretId string,
	checkInterval time.Duration, metricDirectory string,
	logger log.DebugLogger) (<-chan string, error) {
	notificationChannel := make(chan string, 1)
	metrics := &gitMetricsType{
		latencyDistribution: tricorder.NewGeometricBucketer(1, 1e5).
			NewCumulativeDistribution(),
	}
	err := tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"git-pull-latency"), metrics.latencyDistribution,
		units.Millisecond, "latency of git pull calls")
	if err != nil {
		return nil, err
	}
	metrics.lastCommitId, err = setupGitRepository(remoteURL, localDirectory,
		awsSecretId, metrics, logger)
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"last-attempted-git-pull-time"), &metrics.lastAttemptedPullTime,
		units.None, "time of last attempted git pull")
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"last-commit-id"), &metrics.lastCommitId,
		units.None, "commit ID in master branch in  last successful git pull")
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"last-successful-git-pull-time"), &metrics.lastSuccessfulPullTime,
		units.None, "time of last successful git pull")
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"last-notification-time"), &metrics.lastNotificationTime, units.None,
		"time of last git change notification")
	if err != nil {
		return nil, err
	}
	metrics.lastNotificationTime = time.Now()
	notificationChannel <- localDirectory
	go watchGitLoop(localDirectory, checkInterval, metrics, notificationChannel,
		logger)
	return notificationChannel, nil
}

func watchGitLoop(directory string, checkInterval time.Duration,
	metrics *gitMetricsType, notificationChannel chan<- string,
	logger log.DebugLogger) {
	for {
		time.Sleep(checkInterval)
		if commitId, err := gitPull(directory, metrics); err != nil {
			logger.Println(err)
		} else if commitId != metrics.lastCommitId {
			metrics.lastCommitId = commitId
			metrics.lastNotificationTime = time.Now()
			notificationChannel <- directory
		}
	}
}

func watchLocal(directory string, checkInterval time.Duration,
	metricDirectory string, logger log.DebugLogger) (<-chan string, error) {
	if err := checkDirectory(directory); err != nil {
		return nil, err
	}
	var lastNotificationTime time.Time
	err := tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"last-notification-time"), &lastNotificationTime, units.None,
		"time of last notification")
	if err != nil {
		return nil, err
	}
	notificationChannel := make(chan string, 1)
	go watchLocalLoop(directory, checkInterval, &lastNotificationTime,
		notificationChannel, logger)
	return notificationChannel, nil
}

func watchLocalLoop(directory string, checkInterval time.Duration,
	lastNotificationTime *time.Time, notificationChannel chan<- string,
	logger log.DebugLogger) {
	for ; ; time.Sleep(checkInterval) {
		if err := checkDirectory(directory); err != nil {
			logger.Println(err)
		} else {
			*lastNotificationTime = time.Now()
			notificationChannel <- directory
		}
	}
}
