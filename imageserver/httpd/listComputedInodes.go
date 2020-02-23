package httpd

import (
	"bufio"
	"fmt"
	"net/http"
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/html"
)

func (s state) listComputedInodesHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>image %s computed inodes</title>\n", imageName)
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	if image := s.imageDataBase.GetImage(imageName); image == nil {
		fmt.Fprintf(writer, "Image: %s UNKNOWN!\n", imageName)
	} else {
		fmt.Fprintf(writer, "Computed files for image: %s\n", imageName)
		fmt.Fprintln(writer, "</h3>")
		fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
		tw, _ := html.NewTableWriter(writer, true, "Filename", "Data Source")
		// Walk the file-system to leverage stable and friendly sort order.
		listComputedInodes(tw, &image.FileSystem.DirectoryInode, "/")
		fmt.Fprintln(writer, "</table>")
	}
	fmt.Fprintln(writer, "</body>")
}

func listComputedInodes(tw *html.TableWriter,
	directoryInode *filesystem.DirectoryInode, name string) {
	for _, dirent := range directoryInode.EntryList {
		if inode, ok := dirent.Inode().(*filesystem.ComputedRegularInode); ok {
			tw.WriteRow("", "",
				path.Join(name, dirent.Name),
				inode.Source,
			)
		} else if inode, ok := dirent.Inode().(*filesystem.DirectoryInode); ok {
			listComputedInodes(tw, inode, path.Join(name, dirent.Name))
		}
	}
}
