package builder

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/verstr"
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"time"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

func unpackImageAndProcessManifest(client *srpc.Client, manifestDir string,
	unpackImageFunc unpackImageFunction, rootDir string, buildLog *bytes.Buffer,
	logger log.Logger) (manifestType, error) {
	manifestFile := path.Join(manifestDir, "manifest")
	var manifest manifestType
	if err := json.ReadFromFile(manifestFile, &manifest); err != nil {
		return manifestType{}, err
	}
	imageName, err := unpackImageFunc(client, manifest.SourceImage, rootDir,
		logger)
	if err != nil {
		return manifestType{},
			errors.New("error unpacking image: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Source image: %s\n", imageName)
	startTime := time.Now()
	if err := processManifest(manifestDir, rootDir, buildLog); err != nil {
		return manifestType{},
			errors.New("error processing manifest: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Processed manifest in %s\n",
		format.Duration(time.Since(startTime)))
	return manifest, nil
}

func processManifest(manifestDir, rootDir string, buildLog io.Writer) error {
	if err := copyFiles(manifestDir, "files", rootDir, buildLog); err != nil {
		return err
	}
	// Copy in system /etc/resolv.conf
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return err
	}
	defer file.Close()
	err = runInTarget(file, buildLog, rootDir, packagerPathname, "copy-in",
		"/etc/resolv.conf")
	if err != nil {
		return err
	}
	err = runScripts(manifestDir, "pre-install-scripts", rootDir, buildLog)
	if err != nil {
		return err
	}
	if err := installPackages(manifestDir, rootDir, buildLog); err != nil {
		return errors.New("error installing packages: " + err.Error())
	}
	err = copyFiles(manifestDir, "post-install-files", rootDir, buildLog)
	if err != nil {
		return err
	}
	err = runScripts(manifestDir, "scripts", rootDir, buildLog)
	if err != nil {
		return err
	}
	err = copyFiles(manifestDir, "post-scripts-files", rootDir, buildLog)
	if err != nil {
		return err
	}
	fmt.Fprintln(buildLog, "\nCleaning packages:")
	startTime := time.Now()
	err = runInTarget(nil, buildLog, rootDir, packagerPathname, "clean")
	if err != nil {
		return errors.New("error cleaning: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Package clean took: %s\n",
		format.Duration(time.Since(startTime)))
	return clearResolvConf(buildLog, rootDir)
}

func copyFiles(manifestDir, dirname, rootDir string, buildLog io.Writer) error {
	startTime := time.Now()
	sourceDir := path.Join(manifestDir, dirname)
	cf := func(destFilename, sourceFilename string, mode os.FileMode) error {
		return copyFile(destFilename, sourceFilename, mode, len(manifestDir)+1,
			buildLog)
	}
	if err := fsutil.CopyTreeWithCopyFunc(rootDir, sourceDir, cf); err != nil {
		return fmt.Errorf("error copying %s: %s", dirname, err)
	}
	fmt.Fprintf(buildLog, "\nCopied %s tree in %s\n",
		dirname, format.Duration(time.Since(startTime)))
	return nil
}

func copyFile(destFilename, sourceFilename string, mode os.FileMode,
	prefixLength int, buildLog io.Writer) error {
	same, err := fsutil.CompareFiles(destFilename, sourceFilename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if same {
		fmt.Fprintf(buildLog, "Same contents for: %s\n",
			sourceFilename[prefixLength:])
		return nil
	}
	return fsutil.CopyFile(destFilename, sourceFilename, mode)
}

func installPackages(manifestDir, rootDir string, buildLog io.Writer) error {
	lines, err := fsutil.LoadLines(path.Join(manifestDir, "package-list"))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if len(lines) < 1 { // Nothing to do.
		fmt.Fprintln(buildLog, "\nNo packages to install")
		return nil
	}
	fmt.Fprintln(buildLog, "\nUpdating packages:")
	startTime := time.Now()
	err = runInTarget(nil, buildLog, rootDir, packagerPathname, "update")
	if err != nil {
		return errors.New("error updating: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Package update took: %s\n",
		format.Duration(time.Since(startTime)))

	fmt.Fprintln(buildLog, "\nUpgrading packages:")
	startTime = time.Now()
	err = runInTarget(nil, buildLog, rootDir, packagerPathname, "upgrade")
	if err != nil {
		return errors.New("error upgrading: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Package upgrade took: %s\n",
		format.Duration(time.Since(startTime)))

	fmt.Fprintln(buildLog, "\nInstalling packages:", strings.Join(lines, " "))
	startTime = time.Now()
	args := []string{"install"}
	args = append(args, lines...)
	err = runInTarget(nil, buildLog, rootDir, packagerPathname, args...)
	if err != nil {
		return errors.New("error installing: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Package install took: %s\n",
		format.Duration(time.Since(startTime)))
	return nil
}

func runScripts(manifestDir, dirname, rootDir string,
	buildLog io.Writer) error {
	scriptsDir := path.Join(manifestDir, dirname)
	file, err := os.Open(scriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(buildLog, "No %s directory\n", dirname)
			return nil
		}
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	if len(names) < 1 {
		fmt.Fprintln(buildLog, "\nNo scripts to run")
		return nil
	}
	verstr.Sort(names)
	fmt.Fprintln(buildLog, "\nRunning scripts:")
	scriptsStartTime := time.Now()
	tmpDir := path.Join(rootDir, ".scripts")
	if err := os.Mkdir(tmpDir, dirPerms); err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	for _, name := range names {
		if len(name) > 0 && name[0] == '.' {
			continue // Skip hidden paths.
		}
		err := fsutil.CopyFile(path.Join(tmpDir, name),
			path.Join(scriptsDir, name),
			dirPerms)
		if err != nil {
			return err
		}
	}
	for _, name := range names {
		fmt.Fprintf(buildLog, "Running script: %s\n", name)
		startTime := time.Now()
		err := runInTarget(nil, buildLog, rootDir, packagerPathname, "run",
			path.Join("/.scripts", name))
		if err != nil {
			return errors.New("error running script: " + name + ": " +
				err.Error())
		}
		timeTaken := time.Since(startTime)
		fmt.Fprintf(buildLog, "Script: %s took %s\n",
			name, format.Duration(timeTaken))
		fmt.Fprintln(buildLog,
			"=================================================================")
	}
	timeTaken := time.Since(scriptsStartTime)
	fmt.Fprintf(buildLog, "Ran scripts in %s\n", format.Duration(timeTaken))
	return nil
}
