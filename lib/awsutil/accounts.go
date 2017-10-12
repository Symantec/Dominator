package awsutil

import (
	"bufio"
	"os"
	"path"
	"strings"
)

func listAccountNames() ([]string, error) {
	var accountNames []string
	if names, err := listFile("credentials", "aws_access_key_id"); err != nil {
		return nil, err
	} else {
		accountNames = append(accountNames, names...)
	}
	if names, err := listFile("config", "role_arn"); err != nil {
		return nil, err
	} else {
		accountNames = append(accountNames, names...)
	}
	return accountNames, nil
}

func listFile(filename string, identifierKeyName string) ([]string, error) {
	pathname := path.Join(os.Getenv("HOME"), ".aws", filename)
	file, err := os.Open(pathname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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
			fields := strings.Fields(line[1 : len(line)-1])
			if len(fields) == 1 {
				lastAccountName = fields[0]
			} else if len(fields) == 2 && fields[0] == "profile" {
				lastAccountName = fields[1]
			}
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
		if key != identifierKeyName {
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
