package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/awsutil"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
	libtags "github.com/Cloud-Foundations/Dominator/lib/tags"
)

var (
	amiName    = flag.String("amiName", "", "AMI Name property")
	enaSupport = flag.Bool("enaSupport", false,
		"If true, set the Enhanced Networking Adaptor capability flag")
	excludeSearchTags libtags.Tags
	expiresIn         = flag.Duration("expiresIn", time.Hour,
		"Date to set for the ExpiresAt tag")
	ignoreMissingUnpackers = flag.Bool("ignoreMissingUnpackers", false,
		"If true, do not generate an error for missing unpackers")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of imageserver")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber, "Port number of imageserver")
	instanceName = flag.String("instanceName", "ImageUnpacker",
		"The Name tag value for image unpacker instances")
	instanceType = flag.String("instanceType", "t2.medium",
		"Instance type to launch")
	marketplaceImage = flag.String("marketplaceImage",
		"3f8t6t8fp5m9xx18yzwriozxi",
		"Product code (default Debian Jessie amd64)")
	marketplaceLoginName = flag.String("marketplaceLoginName", "admin",
		"Login name for instance booted from marketplace image")
	maxIdleTime = flag.Duration("maxIdleTime", time.Minute*5,
		"Maximum idle time for image unpacker instances")
	minFreeBytes = flag.Uint64("minFreeBytes", 1<<28,
		"minimum number of free bytes in image")
	minImageAge = flag.Duration("minImageAge", time.Hour*24,
		"Minimum image age when listing or deleting unused images")
	oldImageInstancesCsvFile = flag.String("oldImageInstancesCsvFile", "",
		"File to write CSV listing old image instances")
	replaceInstances = flag.Bool("replaceInstances", false,
		"If true, replace old instances when launching, else skip on old")
	rootVolumeSize = flag.Uint("rootVolumeSize", 0,
		"Size of root volume when launching instances")
	s3Bucket = flag.String("s3Bucket", "",
		"S3 bucket to upload bundle to (default is EBS-backed AMIs)")
	s3Folder = flag.String("s3Folder", "",
		"S3 folder to upload bundle to (default is EBS-backed AMIs)")
	searchTags              = libtags.Tags{"Preferred": "true"}
	securityGroupSearchTags libtags.Tags
	sharingAccountName      = flag.String("sharingAccountName", "",
		"Account from which to share AMIs (for S3-backed)")
	skipTargets awsutil.TargetList
	sshKeyName  = flag.String("sshKeyName", "",
		"Name of SSH key for instance")
	subnetSearchTags    libtags.Tags = libtags.Tags{"Network": "Private"}
	tags                libtags.Tags
	targets             awsutil.TargetList
	unusedImagesCsvFile = flag.String("unusedImagesCsvFile", "",
		"File to write CSV listing unused images")
	vpcSearchTags libtags.Tags = libtags.Tags{"Preferred": "true"}
)

func init() {
	flag.Var(&excludeSearchTags, "excludeSearchTags",
		"Name of exclude tags to use when searching for resources")
	flag.Var(&searchTags, "searchTags",
		"Name of tags to use when searching for resources")
	flag.Var(&securityGroupSearchTags, "securityGroupSearchTags",
		"Restrict security group search to given tags")
	flag.Var(&skipTargets, "skipTargets",
		"List of targets to skip (default none). No wildcards permitted")
	flag.Var(&subnetSearchTags, "subnetSearchTags",
		"Restrict subnet search to given tags")
	flag.Var(&tags, "tags", "Tags to apply")
	flag.Var(&targets, "targets",
		"List of targets (default all accounts and regions)")
	flag.Var(&vpcSearchTags, "vpcSearchTags",
		"Restrict VPC search to given tags")
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: ami-publisher [flags...] publish [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"add-volumes", "sizeInGiB", 1, 1, addVolumesSubcommand},
	{"copy-bootstrap-image", "stream-name", 1, 1, copyBootstrapImageSubcommand},
	{"delete", "results-file...", 1, -1, deleteSubcommand},
	{"delete-tags", "tag-key results-file...", 2, -1, deleteTagsSubcommand},
	{"delete-tags-on-unpackers", "tag-key", 1, 1,
		deleteTagsOnUnpackersSubcommand},
	{"delete-unused-images", "", 0, 0, deleteUnusedImagesSubcommand},
	{"expire", "", 0, 0, expireSubcommand},
	{"import-key-pair", "name pub-key-file", 2, 2, importKeyPairSubcommand},
	{"launch-instances", "boot-image", 1, 1, launchInstancesSubcommand},
	{"launch-instances-for-images", "results-file...", 0, -1,
		launchInstancesForImagesSubcommand},
	{"list-images", "", 0, 0, listImagesSubcommand},
	{"list-streams", "", 0, 0, listStreamsSubcommand},
	{"list-unpackers", "", 0, 0, listUnpackersSubcommand},
	{"list-unused-images", "", 0, 0, listUnusedImagesSubcommand},
	{"list-used-images", "", 0, 0, listUsedImagesSubcommand},
	{"prepare-unpackers", "[stream-name]", 0, 1, prepareUnpackersSubcommand},
	{"publish", "image-leaf-name", 2, 2, publishSubcommand},
	{"remove-unused-volumes", "", 0, 0, removeUnusedVolumesSubcommand},
	{"set-exclusive-tags", "key value results-file...", 2, -1,
		setExclusiveTagsSubcommand},
	{"set-tags-on-unpackers", "", 0, 0, setTagsSubcommand},
	{"start-instances", "", 0, 0, startInstancesSubcommand},
	{"stop-idle-unpackers", "", 0, 0, stopIdleUnpackersSubcommand},
	{"terminate-instances", "", 0, 0, terminateInstancesSubcommand},
}

func doMain() int {
	if err := loadflags.LoadForCli("ami-publisher"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmdlogger.SetDatestampsDefault(true)
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 2
	}
	logger := cmdlogger.New()
	if err := setupclient.SetupTls(true); err != nil {
		logger.Println(err)
		return 1
	}
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
