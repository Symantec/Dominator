package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/mbr"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
)

var (
	allocateBlocks = flag.Bool("allocateBlocks", false,
		"If true, allocate blocks when making raw image")
	buildLog = flag.String("buildLog", "",
		"Filename or URL containing build log")
	compress      = flag.Bool("compress", false, "If true, compress tar output")
	computedFiles = flag.String("computedFiles", "",
		"Name of file containing computed files list")
	computedFilesRoot = flag.String("computedFilesRoot", "",
		"Name of directory tree containing computed files to replace on unpack")
	copyMtimesFrom = flag.String("copyMtimesFrom", "",
		"Name of image to copy mtimes for otherwise unchanged files/devices")
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	deleteFilter = flag.String("deleteFilter", "",
		"Name of delete filter file for addi, adds subcommand and right image")
	expiresIn = flag.Duration("expiresIn", 0,
		"How long before the image expires (auto deletes). Default: never")
	filterFile = flag.String("filterFile", "",
		"Filter file to apply when adding images")
	ignoreExpiring = flag.Bool("ignoreExpiring", false,
		"If true, ignore expiring images when finding images")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	makeBootable = flag.Bool("makeBootable", true,
		"If true, make raw image bootable by installing GRUB")
	minFreeBytes = flag.Uint64("minFreeBytes", 4<<20,
		"minimum number of free bytes in raw image")
	releaseNotes = flag.String("releaseNotes", "",
		"Filename or URL containing release notes")
	requiredPaths = flagutil.StringToRuneMap(constants.RequiredPaths)
	roundupPower  = flag.Uint64("roundupPower", 24,
		"power of 2 to round up raw image size")
	skipFields = flag.String("skipFields", "",
		"Fields to skip when showing or diffing images")
	tableType mbr.TableType = mbr.TABLE_TYPE_MSDOS
	timeout                 = flag.Duration("timeout", 0,
		"Timeout for get subcommand")

	logger            log.DebugLogger
	minimumExpiration = 15 * time.Minute
)

func init() {
	flag.Var(&requiredPaths, "requiredPaths",
		"Comma separated list of required path:type entries")
	flag.Var(&tableType, "tableType", "partition table type for make-raw-image")
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: imagetool [flags...] add|check|delete|list [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add    name imagefile filterfile triggerfile")
	fmt.Fprintln(os.Stderr, "  addi   name imagename filterfile triggerfile")
	fmt.Fprintln(os.Stderr, "  adds   name subname filterfile triggerfile")
	fmt.Fprintln(os.Stderr, "  addrep name baseimage layerimage...")
	fmt.Fprintln(os.Stderr, "  bulk-addrep layerimage...")
	fmt.Fprintln(os.Stderr, "  check  name")
	fmt.Fprintln(os.Stderr, "  check-directory dirname")
	fmt.Fprintln(os.Stderr, "  chown  dirname ownerGroup")
	fmt.Fprintln(os.Stderr, "  copy   name oldimagename")
	fmt.Fprintln(os.Stderr, "  delete name")
	fmt.Fprintln(os.Stderr, "  delunrefobj percentage bytes")
	fmt.Fprintln(os.Stderr, "  diff   tool left right")
	fmt.Fprintln(os.Stderr, "         left & right are image sources. Format:")
	fmt.Fprintln(os.Stderr, "         type:name where type is one of:")
	fmt.Fprintln(os.Stderr, "           d: name of directory tree to scan")
	fmt.Fprintln(os.Stderr, "           f: name of file containing a FileSystem")
	fmt.Fprintln(os.Stderr, "           i: name of an image on the imageserver")
	fmt.Fprintln(os.Stderr, "           l: name of file containing an Image")
	fmt.Fprintln(os.Stderr, "           s: name of sub to poll")
	fmt.Fprintln(os.Stderr, "  estimate-usage    name")
	fmt.Fprintln(os.Stderr, "  find-latest-image directory")
	fmt.Fprintln(os.Stderr, "  get               name directory")
	fmt.Fprintln(os.Stderr, "  list")
	fmt.Fprintln(os.Stderr, "  listdirs")
	fmt.Fprintln(os.Stderr, "  listunrefobj")
	fmt.Fprintln(os.Stderr, "  list-latest-image directory")
	fmt.Fprintln(os.Stderr, "  make-raw-image    name rawfile")
	fmt.Fprintln(os.Stderr, "  match-triggers    name triggers-file")
	fmt.Fprintln(os.Stderr, "  merge-filters     filter-file...")
	fmt.Fprintln(os.Stderr, "  merge-triggers    triggers-file...")
	fmt.Fprintln(os.Stderr, "  mkdir             name")
	fmt.Fprintln(os.Stderr, "  show              name")
	fmt.Fprintln(os.Stderr, "  showunrefobj")
	fmt.Fprintln(os.Stderr, "  tar               name [file]")
	fmt.Fprintln(os.Stderr, "Fields:")
	fmt.Fprintln(os.Stderr, "  m: mode")
	fmt.Fprintln(os.Stderr, "  l: number of hardlinks")
	fmt.Fprintln(os.Stderr, "  u: UID")
	fmt.Fprintln(os.Stderr, "  g: GID")
	fmt.Fprintln(os.Stderr, "  s: size/Rdev")
	fmt.Fprintln(os.Stderr, "  t: time of last modification")
	fmt.Fprintln(os.Stderr, "  n: name")
	fmt.Fprintln(os.Stderr, "  d: data (hash or symlink target)")
}

type commandFunc func([]string)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add", 4, 4, addImagefileSubcommand},
	{"adds", 4, 4, addImagesubSubcommand},
	{"addi", 4, 4, addImageimageSubcommand},
	{"addrep", 3, -1, addReplaceImageSubcommand},
	{"bulk-addrep", 1, -1, bulkAddReplaceImagesSubcommand},
	{"check", 1, 1, checkImageSubcommand},
	{"check-directory", 1, 1, checkDirectorySubcommand},
	{"chown", 2, 2, chownDirectorySubcommand},
	{"copy", 2, 2, copyImageSubcommand},
	{"delete", 1, 1, deleteImageSubcommand},
	{"delunrefobj", 2, 2, deleteUnreferencedObjectsSubcommand},
	{"diff", 3, 3, diffSubcommand},
	{"estimate-usage", 1, 1, estimateImageUsageSubcommand},
	{"find-latest-image", 1, 1, findLatestImageSubcommand},
	{"get", 2, 2, getImageSubcommand},
	{"list", 0, 0, listImagesSubcommand},
	{"listdirs", 0, 0, listDirectoriesSubcommand},
	{"listunrefobj", 0, 0, listUnreferencedObjectsSubcommand},
	{"list-latest-image", 1, 1, listLatestImageSubcommand},
	{"make-raw-image", 2, 2, makeRawImageSubcommand},
	{"match-triggers", 2, 2, matchTriggersSubcommand},
	{"merge-filters", 1, -1, mergeFiltersSubcommand},
	{"merge-triggers", 1, -1, mergeTriggersSubcommand},
	{"mkdir", 1, 1, makeDirectorySubcommand},
	{"show", 1, 1, showImageSubcommand},
	{"showunrefobj", 0, 0, showUnreferencedObjectsSubcommand},
	{"tar", 1, 2, tarImageSubcommand},
}

var imageSrpcClient *srpc.Client
var theObjectClient *objectclient.ObjectClient

var listSelector filesystem.ListSelector

func getClients() (*srpc.Client, *objectclient.ObjectClient) {
	if imageSrpcClient == nil {
		var err error
		clientName := fmt.Sprintf("%s:%d",
			*imageServerHostname, *imageServerPortNum)
		imageSrpcClient, err = srpc.DialHTTP("tcp", clientName, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
			os.Exit(1)
		}
		theObjectClient = objectclient.NewObjectClient(clientName)
	}
	return imageSrpcClient, theObjectClient
}

func makeListSelector(arg string) filesystem.ListSelector {
	var mask filesystem.ListSelector = filesystem.ListSelectAll
	for _, char := range arg {
		switch char {
		case 'm':
			mask |= filesystem.ListSelectSkipMode
		case 'l':
			mask |= filesystem.ListSelectSkipNumLinks
		case 'u':
			mask |= filesystem.ListSelectSkipUid
		case 'g':
			mask |= filesystem.ListSelectSkipGid
		case 's':
			mask |= filesystem.ListSelectSkipSizeDevnum
		case 't':
			mask |= filesystem.ListSelectSkipMtime
		case 'n':
			mask |= filesystem.ListSelectSkipName
		case 'd':
			mask |= filesystem.ListSelectSkipData
		}
	}
	return mask
}

var listFilter *filter.Filter

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	logger = cmdlogger.New()
	if *expiresIn > 0 && *expiresIn < minimumExpiration {
		fmt.Fprintf(os.Stderr, "Minimum expiration: %s\n", minimumExpiration)
		os.Exit(2)
	}
	listSelector = makeListSelector(*skipFields)
	var err error
	if *filterFile != "" {
		listFilter, err = filter.Load(*filterFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
