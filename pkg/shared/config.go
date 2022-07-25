package shared

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

var projectRootDirectory = GetProjectRootDir()

// ReadFileValueInt Read the contents of file into integer value
func ReadFileValueInt(file string, val *int) error {
	fileContents, err := ReadFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.Atoi(fileContents)
	if err != nil {
		return fmt.Errorf("reading int value from file: %w", err)
	}
	return nil
}

// ReadFileValueString Read the contents of file into string value
func ReadFileValueString(file string, val *string) error {
	fileContents, err := ReadFile(file)
	if err != nil {
		return err
	}

	*val = strings.TrimSuffix(fileContents, "\n")
	return err
}

// ReadFileValueBool Read the contents of file into boolean value
func ReadFileValueBool(file string, val *bool) error {
	fileContents, err := ReadFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.ParseBool(fileContents)
	if err != nil {
		return fmt.Errorf("reading bool value from file: %w", err)
	}
	return nil
}

// ReadFile ...
func ReadFile(file string) (string, error) {
	absFilePath := BuildFullFilePath(file)

	// If no file is provided then we don't try to read it
	if absFilePath == "" {
		return "", nil
	}

	// Read the file
	buf, err := ioutil.ReadFile(absFilePath)
	if err != nil {
		return "", fmt.Errorf("reading config file: %w", err)
	}
	return string(buf), nil
}

// BuildFullFilePath ...
func BuildFullFilePath(filename string) string {
	// If the value is in quotes, unquote it
	unquotedFile, err := strconv.Unquote(filename)
	if err != nil {
		// values without quotes will raise an error, ignore it.
		unquotedFile = filename
	}

	// If no file is provided, leave val unchanged.
	if unquotedFile == "" {
		return ""
	}

	// Ensure the absolute file path is used
	absFilePath := unquotedFile
	if !filepath.IsAbs(unquotedFile) {
		absFilePath = filepath.Join(projectRootDirectory, unquotedFile)
	}
	return absFilePath
}

// ReadYamlFile ...
func ReadYamlFile(filename string, out interface{}) (err error) {
	fileContents, err := ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), out)
	if err != nil {
		return fmt.Errorf("reading config yaml file: %w", err)
	}
	return nil
}
