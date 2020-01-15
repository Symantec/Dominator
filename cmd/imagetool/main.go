package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/mbr"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
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
		"Name of delete filter file for addi, adds and diff subcommands")
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

	diffArgs = `  tool left right
         left & right are image sources. Format:
         type:name where type is one of:
           d: name of directory tree to scan
           f: name of file containing a FileSystem
           i: name of an image on the imageserver
           l: name of file containing an Image
           s: name of sub to poll`
)

func init() {
	flag.Var(&requiredPaths, "requiredPaths",
		"Comma separated list of required path:type entries")
	flag.Var(&tableType, "tableType", "partition table type for make-raw-image")
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w,
		"Usage: imagetool [flags...] add|check|delete|list [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
	fmt.Fprintln(w, "Fields:")
	fmt.Fprintln(w, "  m: mode")
	fmt.Fprintln(w, "  l: number of hardlinks")
	fmt.Fprintln(w, "  u: UID")
	fmt.Fprintln(w, "  g: GID")
	fmt.Fprintln(w, "  s: size/Rdev")
	fmt.Fprintln(w, "  t: time of last modification")
	fmt.Fprintln(w, "  n: name")
	fmt.Fprintln(w, "  d: data (hash or symlink target)")
}

var subcommands = []commands.Command{
	{"add", "   name imagefile filterfile triggerfile", 4, 4,
		addImagefileSubcommand},
	{"addi", "  name imagename filterfile triggerfile", 4, 4,
		addImageimageSubcommand},
	{"addrep", "name baseimage layerimage...", 3, -1,
		addReplaceImageSubcommand},
	{"adds", "  name subname filterfile triggerfile", 4, 4,
		addImagesubSubcommand},
	{"bulk-addrep", "layerimage...", 1, -1, bulkAddReplaceImagesSubcommand},
	{"change-image-expiration", "name", 1, 1, changeImageExpirationSubcommand},
	{"check", " name", 1, 1, checkImageSubcommand},
	{"check-directory", "dirname", 1, 1, checkDirectorySubcommand},
	{"chown", " dirname ownerGroup", 2, 2, chownDirectorySubcommand},
	{"copy", "  name oldimagename", 2, 2, copyImageSubcommand},
	{"delete", "name", 1, 1, deleteImageSubcommand},
	{"delunrefobj", "percentage bytes", 2, 2,
		deleteUnreferencedObjectsSubcommand},
	{"diff", diffArgs, 3, 3, diffSubcommand},
	{"estimate-usage", "     name", 1, 1, estimateImageUsageSubcommand},
	{"find-latest-image", "  directory", 1, 1, findLatestImageSubcommand},
	{"get", "                name directory", 2, 2, getImageSubcommand},
	{"get-archive-data", "   name outfile", 2, 2,
		getImageArchiveDataSubcommand},
	{"get-file-in-image", "  name imageFile [outfile]", 2, 3,
		getFileInImageSubcommand},
	{"get-image-expiration", "name", 1, 1, getImageExpirationSubcommand},
	{"list", "", 0, 0, listImagesSubcommand},
	{"listdirs", "", 0, 0, listDirectoriesSubcommand},
	{"listunrefobj", "", 0, 0, listUnreferencedObjectsSubcommand},
	{"make-raw-image", "     name rawfile", 2, 2, makeRawImageSubcommand},
	{"match-triggers", "     name triggers-file", 2, 2,
		matchTriggersSubcommand},
	{"merge-filters", "      filter-file...", 1, -1, mergeFiltersSubcommand},
	{"merge-triggers", "     triggers-file...", 1, -1, mergeTriggersSubcommand},
	{"mkdir", "              name", 1, 1, makeDirectorySubcommand},
	{"show", "               name", 1, 1, showImageSubcommand},
	{"showunrefobj", "", 0, 0, showUnreferencedObjectsSubcommand},
	{"tar", "                name [file]", 1, 2, tarImageSubcommand},
	{"test-download-speed", "name", 1, 1, testDownloadSpeedSubcommand},
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
			fmt.Fprintf(os.Stderr, "Error dialing: %s: %s\n", clientName, err)
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

func doMain() int {
	if err := loadflags.LoadForCli("imagetool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 2
	}
	logger = cmdlogger.New()
	if *expiresIn > 0 && *expiresIn < minimumExpiration {
		fmt.Fprintf(os.Stderr, "Minimum expiration: %s\n", minimumExpiration)
		return 2
	}
	listSelector = makeListSelector(*skipFields)
	var err error
	if *filterFile != "" {
		listFilter, err = filter.Load(*filterFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
