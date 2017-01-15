package awsutil

import (
	"bufio"
	"os"
	"path"
	"strings"
)

func listAccountNames() ([]string, error) {
	filename := path.Join(os.Getenv("HOME"), ".aws", "credentials")
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	accountNames := make([]string, 0)
	accessKeyIds := make(map[string]struct{})
	lastAccountName := ""
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}
		if line[0] == '#' {
			continue
		}
		if line[0] == '[' && line[len(line)-1] == ']' {
			lastAccountName = line[1 : len(line)-1]
			continue
		}
		if lastAccountName == "" {
			continue
		}
		splitString := strings.Split(line, "=")
		if len(splitString) != 2 {
			continue
		}
		key := strings.TrimSpace(splitString[0])
		value := strings.TrimSpace(splitString[1])
		if key != "aws_access_key_id" {
			continue
		}
		if _, ok := accessKeyIds[value]; !ok {
			accountNames = append(accountNames, lastAccountName)
			accessKeyIds[value] = struct{}{}
			lastAccountName = ""
		}
	}
	return accountNames, scanner.Err()
}
