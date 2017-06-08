package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
	"io"
	"syscall"
	"time"
)

var statisticsComputeBucketer *tricorder.Bucketer
var statisticsComputeCpuTimeDistribution *tricorder.CumulativeDistribution

func init() {
	statisticsComputeBucketer = tricorder.NewGeometricBucketer(0.1, 100e3)
	statisticsComputeCpuTimeDistribution =
		statisticsComputeBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/statistics-compute-cputime",
		statisticsComputeCpuTimeDistribution,
		units.Millisecond, "statistics compute CPU time")

}

func (imageObjectServers *imageObjectServersType) WriteHtml(writer io.Writer) {
	// TODO(rgooch): These statistics should be cached and the cache invalidated
	//               when images and objects are added/deleted.
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	objectsMap := imageObjectServers.objSrv.ListObjectSizes()
	var totalBytes uint64
	for _, bytes := range objectsMap {
		totalBytes += bytes
	}
	numObjects := len(objectsMap)
	imageObjectServers.objSrv.WriteHtml(writer)
	numUnreferencedObjects, unreferencedBytes :=
		imageObjectServers.imdb.GetUnreferencedObjectsStatistics()
	unreferencedObjectsPercent := 0.0
	if numObjects > 0 {
		unreferencedObjectsPercent =
			100.0 * float64(numUnreferencedObjects) / float64(numObjects)
	}
	unreferencedBytesPercent := 0.0
	if totalBytes > 0 {
		unreferencedBytesPercent =
			100.0 * float64(unreferencedBytes) / float64(totalBytes)
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	statisticsComputeCpuTimeDistribution.Add(time.Duration(
		rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond)
	fmt.Fprintf(writer,
		"Number of unreferenced objects: %d (%.1f%%), "+
			"consuming %s (%.1f%%)<br>\n",
		numUnreferencedObjects, unreferencedObjectsPercent,
		format.FormatBytes(unreferencedBytes), unreferencedBytesPercent)
}
