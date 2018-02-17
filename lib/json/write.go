package json

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	"github.com/Symantec/Dominator/lib/fsutil"
)

func writeToFile(filename string, perm os.FileMode, indent string,
	value interface{}) error {
	file, err := fsutil.CreateRenamingWriter(filename, perm)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	return writeWithIndent(writer, indent, value)
}

func writeWithIndent(w io.Writer, indent string, value interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", indent)
	return encoder.Encode(value)
}
