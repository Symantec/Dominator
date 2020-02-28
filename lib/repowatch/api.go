package repowatch

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

type Config struct {
	AwsSecretId              string        `yaml:"aws_secret_id"`
	Branch                   string        `yaml:"branch"`
	CheckInterval            time.Duration `yaml:"check_interval"`
	LocalRepositoryDirectory string        `yaml:"local_repository_directory"`
	RepositoryURL            string        `yaml:"repository_url"`
}

func Watch(remoteURL, localDirectory string, checkInterval time.Duration,
	metricDirectory string, logger log.DebugLogger) (<-chan string, error) {
	return watch(Config{
		CheckInterval:            checkInterval,
		LocalRepositoryDirectory: localDirectory,
		RepositoryURL:            remoteURL,
	}, metricDirectory, logger)
}

func WatchWithConfig(config Config, metricDirectory string,
	logger log.DebugLogger) (<-chan string, error) {
	return watch(config, metricDirectory, logger)
}
