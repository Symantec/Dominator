package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/html"
	libjson "github.com/Symantec/Dominator/lib/json"
)

const codeStyle = `background-color: #eee; border: 1px solid #999; display: block; float: left;`

func (stream *bootstrapStream) WriteHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Bootstrap command: <code>%s</code><br>\n",
		strings.Join(stream.BootstrapCommand, " "))
	if len(stream.FilterLines) > 0 {
		fmt.Fprintln(writer, "Filter lines:<br>")
		fmt.Fprintf(writer, "<pre style=\"%s\">\n", codeStyle)
		libjson.WriteWithIndent(writer, "    ", stream.FilterLines)
		fmt.Fprintln(writer, "</pre><p style=\"clear: both;\">")
	}
	packager := stream.builder.packagerTypes[stream.PackagerType]
	packager.WriteHtml(writer)
}

func (b *Builder) getHtmlWriter(streamName string) html.HtmlWriter {
	if stream := b.getBootstrapStream(streamName); stream != nil {
		return stream
	}
	if stream := b.getNormalStream(streamName); stream != nil {
		return stream
	}
	// Ensure a nil interface is returned, not a stream with value == nil.
	return nil
}

func (b *Builder) showImageStream(writer io.Writer, streamName string) {
	stream := b.getHtmlWriter(streamName)
	if stream == nil {
		fmt.Fprintf(writer, "<b>Stream: %s does not exist!</b>\n", streamName)
		return
	}
	fmt.Fprintf(writer, "<h3>Information for stream: %s</h3>\n", streamName)
	stream.WriteHtml(writer)
}

func (b *Builder) showImageStreams(writer io.Writer) {
	streamNames := b.listAllStreamNames()
	sort.Strings(streamNames)
	fmt.Fprintln(writer, `<table border="1">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Image Stream</th>")
	fmt.Fprintln(writer, "    <th>ManifestUrl</th>")
	fmt.Fprintln(writer, "    <th>ManifestDirectory</th>")
	fmt.Fprintln(writer, "  </tr>")
	for _, streamName := range streamNames {
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer,
			"    <td><a href=\"showImageStream?%s\">%s</a></td>\n",
			streamName, streamName)
		if imageStream := b.getNormalStream(streamName); imageStream == nil {
			fmt.Fprintln(writer, "    <td></td>")
			fmt.Fprintln(writer, "    <td></td>")
		} else {
			fmt.Fprintf(writer, "    <td>%s</td>\n", imageStream.ManifestUrl)
			fmt.Fprintf(writer, "    <td>%s</td>\n",
				imageStream.ManifestDirectory)
		}
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table><br>")
}

func (b *Builder) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer,
		"Number of image streams: <a href=\"showImageStreams\">%d</a><p>\n",
		b.getNumNormalStreams())
	currentBuilds := make([]string, 0)
	goodBuilds := make(map[string]buildResultType)
	failedBuilds := make(map[string]buildResultType)
	b.buildResultsLock.RLock()
	for name := range b.currentBuildLogs {
		currentBuilds = append(currentBuilds, name)
	}
	for name, result := range b.lastBuildResults {
		if result.error == nil {
			goodBuilds[name] = result
		} else {
			failedBuilds[name] = result
		}
	}
	b.buildResultsLock.RUnlock()
	if len(currentBuilds) > 0 {
		fmt.Fprintln(writer, "Current image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range currentBuilds {
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer,
				"    <td><a href=\"showCurrentBuildLog?%s#bottom\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
	if len(failedBuilds) > 0 {
		streamNames := make([]string, 0, len(failedBuilds))
		for streamName := range failedBuilds {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)
		fmt.Fprintln(writer, "Failed image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Error</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range streamNames {
			result := failedBuilds[streamName]
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer, "    <td>%s</td>\n", result.error)
			fmt.Fprintf(writer,
				"    <td><a href=\"showLastBuildLog?%s\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
	if len(goodBuilds) > 0 {
		streamNames := make([]string, 0, len(goodBuilds))
		for streamName := range goodBuilds {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)
		fmt.Fprintln(writer, "Successful image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Name</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range streamNames {
			result := goodBuilds[streamName]
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer,
				"    <td><a href=\"http://%s/showImage?%s\">%s</a></td>\n",
				b.imageServerAddress, result.imageName, result.imageName)
			fmt.Fprintf(writer,
				"    <td><a href=\"showLastBuildLog?%s\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
}

func (stream *imageStreamType) WriteHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Manifest URL: <code>%s</code><br>\n",
		stream.ManifestUrl)
	fmt.Fprintf(writer, "Manifest Directory: <code>%s</code><br>\n",
		stream.ManifestDirectory)
	buildLog := new(bytes.Buffer)
	manifestDirectory, err := stream.getManifest(stream.builder, stream.name,
		"", buildLog)
	if err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	defer os.RemoveAll(manifestDirectory)
	manifestFilename := path.Join(manifestDirectory, "manifest")
	manifestBytes, err := ioutil.ReadFile(manifestFilename)
	if err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	var manifest manifestType
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	sourceStream := stream.builder.getHtmlWriter(manifest.SourceImage)
	if sourceStream == nil {
		fmt.Fprintf(writer, "SourceImage: <code>%s</code><br>\n",
			manifest.SourceImage)
	} else {
		fmt.Fprintf(writer,
			"SourceImage: <a href=\"showImageStream?%s\"><code>%s</code></a><br>\n",
			manifest.SourceImage, manifest.SourceImage)
	}
	fmt.Fprintln(writer, "Contents of <code>manifest</code> file:<br>")
	fmt.Fprintf(writer, "<pre style=\"%s\">\n", codeStyle)
	writer.Write(manifestBytes)
	fmt.Fprintln(writer, "</pre><p style=\"clear: both;\">")
	packagesFile, err := os.Open(path.Join(manifestDirectory, "package-list"))
	if err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	defer packagesFile.Close()
	fmt.Fprintln(writer, "Contents of <code>package-list</code> file:<br>")
	fmt.Fprintf(writer, "<pre style=\"%s\">\n", codeStyle)
	io.Copy(writer, packagesFile)
	fmt.Fprintln(writer, "</pre><p style=\"clear: both;\">")
	if size, err := getTreeSize(manifestDirectory); err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	} else {
		fmt.Fprintf(writer, "Manifest tree size: %s<br>\n",
			format.FormatBytes(size))
	}
	fmt.Fprintln(writer, "<hr style=\"height:2px\"><font color=\"#bbb\">")
	fmt.Fprintln(writer, "<b>Logging output:</b>")
	fmt.Fprintln(writer, "<pre>")
	io.Copy(writer, buildLog)
	fmt.Fprintln(writer, "</pre>")
	fmt.Fprintln(writer, "</font>")
}

func (packager *packagerType) WriteHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Clean command: <code>%s</code><br>\n",
		strings.Join(packager.CleanCommand, " "))
	fmt.Fprintf(writer, "Install command: <code>%s</code><br>\n",
		strings.Join(packager.InstallCommand, " "))
	fmt.Fprintf(writer, "List command: <code>%s</code><br>\n",
		strings.Join(packager.ListCommand.ArgList, " "))
	if packager.ListCommand.SizeMultiplier > 1 {
		fmt.Fprintf(writer, "List command size multiplier: %d<br>\n",
			packager.ListCommand.SizeMultiplier)
	}
	fmt.Fprintf(writer, "Update command: <code>%s</code><br>\n",
		strings.Join(packager.UpdateCommand, " "))
	fmt.Fprintf(writer, "Upgrade command: <code>%s</code><br>\n",
		strings.Join(packager.UpgradeCommand, " "))
	if len(packager.Verbatim) > 0 {
		fmt.Fprintln(writer, "Verbatim lines:<br>")
		fmt.Fprintf(writer, "<pre style=\"%s\">\n", codeStyle)
		libjson.WriteWithIndent(writer, "    ", packager.Verbatim)
		fmt.Fprintln(writer, "</pre><p style=\"clear: both;\">")
	}
	fmt.Fprintln(writer, "Package installer script:<br>")
	fmt.Fprintf(writer, "<pre style=\"%s\">\n", codeStyle)
	packager.writePackageInstallerContents(writer)
	fmt.Fprintln(writer, "</pre><p style=\"clear: both;\">")
}
