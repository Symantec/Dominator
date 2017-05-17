package httpd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/html"
	"io"
	"net/http"
	"sort"
)

func (s state) statusHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>image-unpacker status page</title>")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>image-unpacker status page</h1>")
	fmt.Fprintln(writer, "</center>")
	html.WriteHeaderWithRequest(writer, req)
	fmt.Fprintln(writer, "<h3>")
	s.writeDashboard(writer)
	for _, htmlWriter := range htmlWriters {
		htmlWriter.WriteHtml(writer)
	}
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}

func (s state) writeDashboard(writer io.Writer) {
	status := s.unpacker.GetStatus()
	fmt.Fprintln(writer, "Image streams:<br>")
	fmt.Fprintln(writer, `<table border="1">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Image Stream</th>")
	fmt.Fprintln(writer, "    <th>Device Id</th>")
	fmt.Fprintln(writer, "    <th>Device Name</th>")
	fmt.Fprintln(writer, "    <th>Size</th>")
	fmt.Fprintln(writer, "    <th>Status</th>")
	fmt.Fprintln(writer, "  </tr>")
	streamNames := make([]string, 0, len(status.ImageStreams))
	for streamName := range status.ImageStreams {
		streamNames = append(streamNames, streamName)
	}
	sort.Strings(streamNames)
	for _, streamName := range streamNames {
		stream := status.ImageStreams[streamName]
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer,
			"    <td><a href=\"showFileSystem?%s\">%s</a></td>\n",
			streamName, streamName)
		fmt.Fprintf(writer, "    <td>%s</td>\n", stream.DeviceId)
		fmt.Fprintf(writer, "    <td>%s</td>\n",
			status.Devices[stream.DeviceId].DeviceName)
		size := status.Devices[stream.DeviceId].Size
		sizeString := format.FormatBytes(size)
		if size < 1 {
			sizeString = ""
		}
		fmt.Fprintf(writer, "    <td>%s</td>\n", sizeString)
		fmt.Fprintf(writer, "    <td>%s</td>\n", stream.Status)
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table><br>")
	fmt.Fprintln(writer, "Devices:<br>")
	fmt.Fprintln(writer, `<table border="1">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Device Id</th>")
	fmt.Fprintln(writer, "    <th>Device Name</th>")
	fmt.Fprintln(writer, "    <th>Size</th>")
	fmt.Fprintln(writer, "    <th>Image Stream</th>")
	fmt.Fprintln(writer, "    <th>Status</th>")
	fmt.Fprintln(writer, "  </tr>")
	deviceIds := make([]string, 0, len(status.Devices))
	for deviceId := range status.Devices {
		deviceIds = append(deviceIds, deviceId)
	}
	sort.Strings(deviceIds)
	for _, deviceId := range deviceIds {
		deviceInfo := status.Devices[deviceId]
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer, "    <td>%s</td>\n", deviceId)
		fmt.Fprintf(writer, "    <td>%s</td>\n", deviceInfo.DeviceName)
		fmt.Fprintf(writer, "    <td>%s</td>\n",
			format.FormatBytes(deviceInfo.Size))
		streamName := deviceInfo.StreamName
		if stream, ok := status.ImageStreams[streamName]; ok {
			fmt.Fprintf(writer,
				"    <td><a href=\"showFileSystem?%s\">%s</a></td>\n",
				streamName, streamName)
			fmt.Fprintf(writer, "    <td>%s</td>\n", stream.Status)
		} else {
			fmt.Fprintln(writer, "    <td></td>")
			fmt.Fprintln(writer, "    <td></td>")
		}
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table><br>")
}
