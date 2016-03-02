package httpd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"io"
	"net/http"
	"path"
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
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Filename</th>")
		fmt.Fprintln(writer, "    <th>Data Source</th>")
		fmt.Fprintln(writer, "  </tr>")
		// Walk the file-system to leverage stable and friendly sort order.
		listComputedInodes(writer, &image.FileSystem.DirectoryInode, "/")
		fmt.Fprintln(writer, "</table>")
	}
	fmt.Fprintln(writer, "</body>")
}

func listComputedInodes(writer io.Writer,
	directoryInode *filesystem.DirectoryInode, name string) {
	for _, dirent := range directoryInode.EntryList {
		if inode, ok := dirent.Inode().(*filesystem.ComputedRegularInode); ok {
			fmt.Fprintln(writer, "  <tr>")
			fmt.Fprintf(writer, "    <td>%s</td>\n",
				path.Join(name, dirent.Name))
			fmt.Fprintf(writer, "    <td>%s</td>\n", inode.Source)
			fmt.Fprintln(writer, "  </tr>")
		} else if inode, ok := dirent.Inode().(*filesystem.DirectoryInode); ok {
			listComputedInodes(writer, inode, path.Join(name, dirent.Name))
		}
	}
}
