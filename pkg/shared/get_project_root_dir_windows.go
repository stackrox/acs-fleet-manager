//go::build windows

package shared

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

// GetProjectRootDir returns the root directory of the project.
// The root directory of the project is the directory that contains the go.mod file which contains
// the "github.com/stackrox/acs-fleet-manager" module name.
func GetProjectRootDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		glog.Fatal(err)
	}

	dirs := strings.Split(workingDir, `\`)

	var goModPath string
	var rootPath string
	for i, d := range dirs {
		if i == 0 {
			rootPath = d + `\`
		} else {
			rootPath = filepath.Join(rootPath, d)
		}
		goModPath = filepath.Join(rootPath, "go.mod")
		goModFile, err := ioutil.ReadFile(goModPath)
		if err != nil { // if the file doesn't exist, continue searching
			continue
		}
		// The project root directory is obtained based on the assumption that module name,
		// "github.com/stackrox/acs-fleet-manager", is contained in the 'go.mod' file.
		// Should the module name change in the code repo then it needs to be changed here too.
		if strings.Contains(string(goModFile), "github.com/stackrox/acs-fleet-manager") {
			break
		}
	}
	return rootPath
}
