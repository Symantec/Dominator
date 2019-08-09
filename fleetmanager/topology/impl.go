package topology

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

var gitPullLatencyDistribution = tricorder.NewGeometricBucketer(1, 1e5).
	NewCumulativeDistribution()

func gitCommand(repositoryDirectory string, command ...string) error {
	cmd := exec.Command("git", command...)
	cmd.Dir = repositoryDirectory
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.New(err.Error() + ": " + string(output))
	}
	return nil
}

func gitPull(repositoryDirectory string) (string, error) {
	startTime := time.Now()
	if err := gitCommand(repositoryDirectory, "pull"); err != nil {
		return "", err
	}
	gitPullLatencyDistribution.Add(time.Since(startTime))
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

func setupGitRepository(topologyRepository, localRepositoryDir string,
	topologyDir string) (string, error) {
	if err := os.MkdirAll(localRepositoryDir, dirPerms); err != nil {
		return "", err
	}
	gitSubdir := filepath.Join(localRepositoryDir, ".git")
	if _, err := os.Stat(gitSubdir); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		err := gitCommand(localRepositoryDir, "clone", topologyRepository, ".")
		if err != nil {
			return "", err
		}
		return readLatestCommitId(localRepositoryDir)
	} else {
		_, err := os.Stat(filepath.Join(localRepositoryDir, topologyDir))
		if err != nil {
		}
		return gitPull(localRepositoryDir)
	}
}

func watch(topologyRepository, localRepositoryDir, topologyDir string,
	checkInterval time.Duration,
	logger log.DebugLogger) (<-chan *Topology, error) {
	topologyChannel := make(chan *Topology, 1)
	if topologyRepository != "" {
		err := tricorder.RegisterMetric("fleet-manager/git-pull-latency",
			gitPullLatencyDistribution, units.Millisecond,
			"latency of git pull calls")
		if err != nil {
			return nil, err
		}
		commitId, err := setupGitRepository(topologyRepository,
			localRepositoryDir, topologyDir)
		if err != nil {
			return nil, err
		}
		go watchLoopGit(localRepositoryDir, topologyDir, commitId,
			checkInterval, topologyChannel, logger)
	} else {
		go watchLoopLocal(topologyDir, checkInterval, topologyChannel, logger)
	}
	return topologyChannel, nil
}

func watchLoopGit(localRepositoryDir, topologySubdir, lastCommitId string,
	checkInterval time.Duration, topologyChannel chan<- *Topology,
	logger log.DebugLogger) {
	topologyDir := filepath.Join(localRepositoryDir, topologySubdir)
	prevTopology, err := load(topologyDir)
	if err != nil {
		logger.Println(err)
	} else {
		topologyChannel <- prevTopology
	}
	time.Sleep(checkInterval)
	for ; ; time.Sleep(checkInterval) {
		if commitId, err := gitPull(localRepositoryDir); err != nil {
			logger.Println(err)
		} else if commitId != lastCommitId {
			lastCommitId = commitId
			if topology, err := load(topologyDir); err != nil {
				logger.Println(err)
			} else {
				if prevTopology.equal(topology) {
					logger.Println("Ignoring unchanged configuration")
				} else {
					topologyChannel <- topology
					prevTopology = topology
				}
			}
		}
	}
}

func watchLoopLocal(topologyDir string, checkInterval time.Duration,
	topologyChannel chan<- *Topology, logger log.DebugLogger) {
	var prevTopology *Topology
	for ; ; time.Sleep(checkInterval) {
		if topology, err := load(topologyDir); err != nil {
			logger.Println(err)
		} else {
			if prevTopology.equal(topology) {
				logger.Debugln(1, "Ignoring unchanged configuration")
			} else {
				topologyChannel <- topology
				prevTopology = topology
			}
		}
	}
}

func (subnet *Subnet) shrink() {
	subnet.Subnet.Shrink()
	subnet.FirstAutoIP = hyper_proto.ShrinkIP(subnet.FirstAutoIP)
	subnet.LastAutoIP = hyper_proto.ShrinkIP(subnet.LastAutoIP)
	for index, ip := range subnet.ReservedIPs {
		if len(ip) == 16 {
			ip = ip.To4()
			if ip != nil {
				subnet.ReservedIPs[index] = ip
			}
		}
	}
}
