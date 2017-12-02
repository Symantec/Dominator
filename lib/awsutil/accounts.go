package awsutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	kDefault = "default"
)

func listAccountNames(options *CredentialsOptions) ([]string, error) {
	var accountNames []string
	if names, err := listFile(options.CredentialsPath, "aws_access_key_id"); err != nil {
		return nil, err
	} else {
		for _, name := range names {
			if name != kDefault {
				accountNames = append(accountNames, name)
			}
		}
	}
	if names, err := listFile(options.ConfigPath, "role_arn"); err != nil {
		return nil, err
	} else {
		for _, name := range names {
			if name != kDefault {
				accountNames = append(accountNames, name)
			}
		}
	}
	return accountNames, nil
}

func listFile(pathname string, identifierKeyName string) ([]string, error) {
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
		} else {
			return nil, fmt.Errorf("Duplicate key identifier found: '%s'", line)
		}
	}
	return accountNames, scanner.Err()
}
