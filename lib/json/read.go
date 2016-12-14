package json

import (
	"bufio"
	"encoding/json"
	"os"
)

func readFromFile(filename string, value interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(bufio.NewReader(file))
	if err := decoder.Decode(value); err != nil {
		return err
	}
	return nil
}
