package errors

import "errors"

func New(text string) error {
	if text == "" {
		return nil
	}
	return errors.New(text)
}

func ErrorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
