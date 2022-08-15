// Code generated by go-bindata. (@generated) DO NOT EDIT.

//Package generated generated by go-bindata.// sources:
// .generate/openapi/fleet-manager.yaml
package generated

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// ModTime return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _fleetManagerYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x7d\x7b\x73\xdb\xb6\xf2\xe8\xff\xfe\x14\x7b\xd5\x7b\xc6\xa7\x1d\x4b\x96\xe4\x47\x12\xcd\xed\x9d\x71\x12\x27\x71\x9b\x38\x8e\xed\x34\x4d\x3b\x67\x64\x88\x84\x24\xd8\x24\x40\x03\xa0\x6d\xe5\x9e\xfb\xdd\x7f\x03\x80\x0f\x90\x04\x29\xca\xce\xc3\x3e\xc7\x9e\xe9\xa4\x22\x81\xe5\xee\x62\xb1\xd8\x05\x76\x17\x2c\xc2\x14\x45\x64\x04\x5b\xbd\x7e\xaf\x0f\x3f\x01\xc5\xd8\x07\x39\x27\x02\x90\x80\x29\xe1\x42\x42\x40\x28\x06\xc9\x00\x05\x01\xbb\x06\xc1\x42\x0c\x07\x2f\xf7\x85\x7a\x74\x41\xd9\xb5\x69\xad\x3a\x50\x48\xc0\x81\xcf\xbc\x38\xc4\x54\xf6\xd6\x7e\x82\xbd\x20\x00\x4c\xfd\x88\x11\x2a\x05\xf8\x78\x4a\x28\xf6\x61\x8e\x39\x86\x6b\x12\x04\x30\xc1\xe0\x13\xe1\xb1\x2b\xcc\xd1\x24\xc0\x30\x59\xa8\x2f\x41\x2c\x30\x17\x3d\x38\x98\x82\xd4\x6d\xd5\x07\x12\xec\x18\x5c\x60\x1c\x19\x4c\x72\xc8\x9d\x88\x93\x2b\x24\x71\x67\x03\x90\xaf\x68\xc0\xa1\x6a\x2a\xe7\x18\x3a\x21\xa2\x68\x86\xfd\xae\xc0\xfc\x8a\x78\x58\x74\x51\x44\xba\x49\xfb\xde\x02\x85\x41\x07\xa6\x24\xc0\x6b\x84\x4e\xd9\x68\x0d\x40\x12\x19\xe0\x11\x1c\x63\x1f\xde\x20\x09\x7b\xfe\x15\xa2\x1e\xf6\xe1\x45\x10\x0b\x89\x39\x9c\x60\x2f\xe6\x44\x2e\xe0\xc4\x00\x84\x57\x01\xc6\x12\xde\xe9\xcf\xf0\x35\x80\x2b\xcc\x05\x61\x74\x04\x83\xde\xb0\xd7\x5f\x03\xf0\xb1\xf0\x38\x89\xa4\x7e\xb8\x1c\xee\x3f\x8f\xdf\xec\xbd\x38\xf9\xd9\x0d\xdf\xf0\xe2\x18\x0b\x09\x7b\x47\x07\x8a\x48\x43\x1f\x10\x2a\xa4\x02\x28\x80\x4d\x61\xef\xc5\x09\x78\x2c\x8c\x18\xc5\x54\x8a\xde\x9a\xa2\x1d\x73\xa1\xc8\xeb\x42\xcc\x83\x11\xcc\xa5\x8c\xc4\x68\x73\x13\x45\xa4\xa7\x46\x4e\xcc\xc9\x54\xf6\x3c\x16\xae\x01\x94\x30\x7e\x87\x08\x85\x7f\x46\x9c\xf9\xb1\xa7\x9e\xfc\x0c\x06\x9c\x1b\x98\x90\x68\x86\x97\x81\x3c\x91\x68\x46\xe8\xcc\x09\x68\xb4\xb9\x19\x30\x0f\x05\x73\x26\xe4\xe8\x69\xbf\xdf\xaf\x76\xcf\xde\xe7\x3d\x37\xab\xad\xbc\x98\x73\x4c\x25\xf8\x2c\x44\x84\xae\x45\x48\xce\x35\x07\x14\x9a\x9b\x7c\x8e\x3c\xb1\x79\x35\x18\xe9\x7e\x33\x2c\xcd\xff\x80\x12\x63\x8e\x14\x80\x03\x7f\xa4\x9e\xff\x61\x46\xf3\x1d\x96\xc8\x47\x12\x25\xad\x38\x16\x11\xa3\x02\x8b\xb4\x1b\x40\x67\xd8\xef\x77\xf2\x9f\x00\x1e\xa3\x12\x53\x69\x3f\x02\x40\x51\x14\x10\x4f\x7f\x60\xf3\x5c\x30\x5a\x7c\x0b\x20\xbc\x39\x0e\x51\xf9\x29\xc0\xff\xe6\x78\x3a\x82\xce\x4f\x9b\xf9\xb0\x6e\x9a\xb6\x62\xb3\x84\x62\xc7\xea\x5c\x60\x48\xd2\x0e\xc2\x22\x2d\x22\x0e\x43\xc4\x17\x4a\x34\x65\xcc\xa9\xd0\xd3\xe6\xaa\xdc\xb6\xcc\xb8\x4d\xcc\x39\xe3\x62\xf3\xff\x11\xff\xff\x2f\x65\xe2\xbe\x6a\xfb\x7c\x71\xe0\xdf\x47\xf6\x69\xe4\x6a\x99\xf6\x1a\x4b\xd0\xa4\x2a\xe5\x94\x11\xe0\xe4\x59\xd6\x8c\xa4\xcd\x24\x9a\x59\x24\x76\x4d\x0b\x91\x3c\x88\x10\x47\x21\x96\xc9\xbc\x4c\x9b\xb8\x30\xcd\x5b\x6e\x12\xbf\x53\x37\x14\xed\x46\x41\xdc\xdb\x21\x78\x4b\x84\xac\x1d\x06\xf5\x52\x69\xb6\x88\x09\x41\xd4\x52\x51\x60\xa5\x73\x38\x82\x72\x17\xa5\x30\x0b\xdd\x6a\x86\xa7\xc2\x5f\x21\x91\x8c\x97\xf3\x37\x51\xd8\x27\xba\xf5\x7d\x64\x73\x01\xc1\x5a\x56\xbf\xbf\xc8\x51\xdd\x29\xa1\x5a\x68\xf8\x91\xe2\x9b\x08\x7b\x12\xfb\x89\xe8\x33\x4f\xeb\x5c\xff\x5e\xcc\x62\xf5\x87\x6f\x50\x18\x05\x36\xf3\xd3\xbf\x9d\x7e\x7f\xdf\xbc\xac\xbe\x73\x7f\x28\x85\xb5\x99\x77\xed\x34\x89\x9f\x11\x1a\x25\x80\x1c\x0b\x16\x73\x0f\x8b\x0d\x10\xb1\x37\x57\xd6\xd5\xf5\x1c\x2b\xd3\x06\x42\x74\x43\xc2\x38\x84\xc4\x38\x01\x0f\x45\xc8\x53\x46\xc0\x1c\x09\x98\x60\x4c\x81\x63\xe4\xcd\x33\x96\x8a\xc4\x48\xb0\xa5\xf6\x39\x46\x1c\xf3\x11\xfc\xfd\xaf\x8a\xe0\x7a\x98\x4a\x8e\x82\x96\x5a\xfa\x85\x69\x6d\xe9\xe9\xc2\x70\x9f\x2a\x5b\x2f\xeb\xa3\x0c\x11\x46\x83\x05\xa0\x58\xce\x19\x27\x5f\x8c\x75\xa6\x4d\x37\x20\xd4\xb0\x00\x85\x18\x18\x9f\x21\x4a\x84\xe9\x84\x0c\x6f\xd8\x35\xc5\xbc\xf8\x86\x4d\x4d\x97\x08\x7b\x64\x4a\x94\x5d\x64\xb0\xe9\xdd\xc7\x89\x94\xe0\x76\x8c\x2f\x63\x5c\x54\x5a\xcd\x52\x57\xec\xf7\x1a\xcb\xe3\x84\xaa\xdb\xca\x62\x11\x60\x49\x2c\x5b\x7c\xf7\x13\x91\xf3\x57\x88\x04\xd8\x7f\xc1\xb1\xe6\x91\x51\x0e\x5f\x07\x9f\x06\xc8\xb5\xda\x27\x81\x00\xdc\x80\x80\x29\x8b\xa9\xaf\xd7\xde\x97\xf9\xc0\x6f\xf7\x07\xf7\xc4\x56\x68\x1e\xef\xed\xfe\xe0\xb6\x9c\xcc\xbb\xd6\xb2\x6a\x2f\x96\x73\x90\xec\x02\xeb\xc9\x48\xe8\x15\x0a\x88\x6f\x33\x69\xeb\x81\x30\x69\xeb\xf6\x4c\xda\x5a\xc6\xa4\x8f\x02\x73\xa0\x4c\x96\xf4\x14\xf2\x3c\x2c\x12\x45\x6d\x74\xaf\xcd\xb8\xed\x07\xc2\xb8\xed\xdb\x33\x6e\x7b\x19\xe3\x0e\x59\x65\x2e\x5e\x13\x39\xb7\x34\xf4\xc1\x4b\xc0\x37\x44\x48\x51\x6f\x2f\xdc\x57\xd6\x7d\xd5\xe5\xbf\xc2\xba\x65\x86\xd1\xd2\x55\x1c\x5c\x46\x05\xaa\x8c\x47\xae\x15\x7d\x1c\x60\x89\x9d\x0b\xbb\x79\xb5\x64\x6d\xff\x77\x86\xc9\xa9\x5a\x9e\xd5\xba\x6e\x56\x72\x6b\xd6\x4c\x19\x37\xfb\x3d\xb9\x0d\x80\xb8\xc5\xbf\xc1\xcf\xba\x33\xf2\x43\x42\x89\x90\x1c\x49\x45\xf9\xf4\xb6\x0b\x3e\xc0\xd0\x00\x34\x7d\x15\x3a\x1b\x80\xa8\x6f\xb0\x23\x53\x20\x52\x6f\x86\x04\x82\x29\x57\x4a\xde\xe1\x53\x6e\x4f\x8c\xd0\x11\x5c\xc6\x98\x2f\xac\x61\xa6\x28\xc4\x23\x40\x62\x41\xbd\xba\xc1\x3f\xc2\x7c\xca\x78\xa8\xbf\x88\x3c\x63\x2a\x51\x40\xd4\xf4\x9a\x73\x46\x59\x2c\x20\x44\x94\xea\x9d\x8f\x26\xa1\x97\x8b\x08\x8f\x60\xc2\x58\x80\x11\xb5\xde\xa8\xf1\x27\x1c\xfb\x23\x90\x3c\xc6\x8d\x06\xd2\xb0\xde\x7c\x7f\xa9\x05\xa3\xb0\x60\x3c\x8c\xc9\xbb\xdd\xef\x6b\xdc\x09\xa3\xb7\xd7\x7f\x65\x10\xf5\xbb\x26\x6a\x55\x35\x72\x64\xfc\xc3\xaa\x9b\xf3\x68\x8f\x3c\xda\x23\x8f\xf6\x88\xb2\x47\x8c\x4e\xb9\x83\x55\x52\x00\xf0\x5f\x6b\x9b\xdc\x8d\x8d\x65\x00\xb7\xb7\x53\x52\x13\xc4\x80\x6b\x36\x41\x5a\x99\x35\xd5\x95\xb6\xd5\x8e\x67\xdd\xbe\x86\x01\x12\x31\xe1\xde\xd3\xf0\x94\xe7\x99\x9a\x3e\x2e\xb3\x67\x1f\x79\x73\x48\x80\xe9\x2d\x17\x04\x82\xd0\x59\xe0\xb4\x22\x94\xed\x51\x7a\xaf\x8c\x92\x1e\x68\x07\x17\x9b\x33\xaa\xeb\x8c\x43\x72\x8e\xb4\x81\xa2\x5a\x6a\x07\x56\xcd\x6d\xd5\xc1\x18\x31\x05\xc8\xb1\x9c\x63\x2a\x95\xdc\x65\x76\x16\x4e\x59\xfc\x1f\x66\xa4\x68\x9a\x9e\x33\xdf\x92\x12\xa7\xff\x6f\x1d\x50\x38\xa7\x6a\xf3\x44\x75\x4f\xd3\xf6\x5b\x3a\x47\x68\x11\x30\xe4\x17\x27\x6d\xdd\x94\xfd\x78\x72\x8c\x67\xa4\xaa\x2b\x96\x4c\xd3\xb4\x5b\xcd\xae\xcd\xfe\xc7\x5b\x41\x4d\xbb\x55\xa0\xde\xda\x68\x7c\x60\xbb\x6a\x47\x4c\x7c\xfb\x6d\xb5\xa2\xdd\xe3\x79\x38\x7a\xa8\x96\x74\xba\x3b\x77\x07\x4b\xba\x04\xe2\xd1\x92\x7e\xb4\xa4\x53\x26\x7d\x65\x4b\x3a\x03\xfb\x0e\xdd\xec\x05\x01\xbb\xc6\xfe\x41\x12\xf7\x70\x6c\xce\x49\xee\xf0\xbd\x65\x30\x9d\x88\x9c\x62\x1e\x8a\x43\x26\x53\x1d\x70\x87\xef\xd7\x80\x6a\xf6\x24\xa6\x8c\x4f\x88\xef\x63\x0a\x98\xe8\x13\xa5\x09\xf6\x50\x2c\x70\x6e\x6d\x10\xd1\xca\xdd\x00\x56\xec\x9b\x9e\x4c\xd1\x38\x9c\x60\xbd\x8f\x93\x47\x98\x68\xd3\xc6\x43\x14\x26\x38\xb1\xb1\x12\x03\x87\x08\xf3\xcd\xf2\xe9\x55\xef\x41\x3a\x33\xdf\x70\x73\xf5\x34\xb7\xef\xb0\x9f\x1d\x10\x82\xcf\xb0\xa0\xeb\xd2\xb8\x2e\x36\xcf\x9e\x3d\x10\x9e\x3d\x3b\x44\x21\x7e\xc1\xe8\x34\x20\x9e\xbc\x3d\xff\x5c\x60\xea\x95\xa5\xe2\x87\x6e\x99\xcb\x9d\x8f\xa5\xf1\x6b\x92\x93\x48\x2f\x59\xa2\xcc\x56\x20\x11\x19\xcb\x1f\xa4\x7b\xf8\x0d\xb7\xae\xf7\x28\xc4\x75\x5e\x21\x5c\xcf\x49\x90\xf2\x92\xce\x34\x63\x4b\xfe\x60\x7b\x4f\xd0\xf2\x2e\x73\xff\xc9\x05\xcd\x3a\xb0\x76\x6c\x89\xa7\x41\x1e\xa5\x9e\xc2\xe5\xed\xbd\xa7\xc1\x02\x78\x76\x44\xcf\x04\x4e\x7d\xbf\x44\xa5\x21\x8e\x8b\xee\x9a\x6b\x17\xd9\xb8\x70\x6d\x3c\xb6\x9a\xf3\x75\xb1\x0a\x93\xda\x1c\x7b\x97\x66\xc3\x12\x96\xc0\x8f\x33\xe9\xcb\x11\x3e\xb0\x82\x59\xaf\xfa\x7e\x1d\x73\xde\x82\xd4\xa9\x37\xd9\x0b\x4c\x7d\x8e\xfc\x92\x84\x7f\x57\x2e\xae\xa8\x21\x0e\x8c\xbd\xf8\x21\xc6\x7c\x71\x07\xb3\xde\x01\xa6\x53\x6f\xa8\xaf\x60\xbf\xde\x5f\xce\x7d\x65\xab\xfe\xbf\xd7\x50\xbf\xeb\x96\xf7\x63\xdc\x59\x9b\xd5\xfb\x56\x11\xa4\x11\x9a\x59\x43\xb5\xb4\xb9\x20\x5f\x56\x69\xce\xb8\x8f\xf9\xf3\xc5\x2a\x1f\xc0\x88\x7b\x73\xc7\x1e\x6f\xc0\x62\x7f\x1c\x71\x76\x45\x7c\xec\x88\x6e\x6d\x8c\xf9\x14\x71\x14\x31\xae\x24\x44\x83\x81\x0c\x4c\xdd\xd2\xac\x5a\x1d\x95\x1a\x7d\x9b\x05\xda\xa0\x8b\xfd\xd6\xb8\xc2\x77\x5d\xb0\x6d\x46\x14\xd7\xeb\x47\x95\xdf\x46\xe5\x3f\x6a\xae\xfb\xa6\xb9\x1a\xd5\x8a\x8e\x8c\xdd\xe4\x7a\xcb\xfc\xd6\x3a\x26\xe9\x9e\xc5\x99\xd4\x4c\xe8\x36\xba\xc7\x6c\xde\xdf\x13\x0d\x94\x12\xf6\xc3\x14\x91\xe1\xc6\xa3\x1a\x7a\x54\x43\xd9\xdf\x8f\x57\x43\xc4\x5f\x41\x09\x7d\x5b\x6b\x2b\xdd\x92\x1d\xcb\x45\xa4\xfb\xfd\xa4\xfe\x83\xd3\x39\x16\x58\xef\x66\xa4\x1b\x16\xdd\x29\xf2\x08\x9d\x01\xc7\x81\xde\xb1\xc8\x52\x27\x93\x3e\x0d\x69\x03\x9b\x21\x96\x9c\x78\x62\x53\x1f\x33\x8f\x39\xa2\x33\xbc\x5c\x51\x26\x9d\x4c\x58\x86\x24\x21\x16\x98\x13\x2c\x40\x77\x37\x27\xd6\x30\x59\x64\x7b\x14\x59\x0c\x41\x59\x39\xbe\x33\x70\x9e\x2f\x8e\x55\xc7\x0f\xd6\x49\xf7\x37\xd6\x8d\xbf\x9d\xbc\x3f\x04\xc4\x39\x5a\x28\x0d\x79\xc4\x59\x88\xe5\x1c\xc7\x39\x65\x6c\x72\x8e\x3d\x29\x60\xca\x59\x08\x6c\xa2\x5c\x18\x24\x19\x27\x71\xf8\x23\x66\x51\xc2\xa7\x9c\x4b\x8f\x4a\xf3\x51\x69\x66\x7f\x0f\x4c\x69\xfa\xb1\xd1\x01\x2b\x69\x42\xa9\x26\x60\xb0\x42\x97\x29\x09\xd4\xbf\xf5\xd1\x45\x0e\xf5\xb7\xa2\xe2\x33\xfa\x59\xde\x46\xdf\x99\x33\x48\xf9\xa8\xf1\x96\x68\x3c\x9b\x4f\x8f\x3a\xef\x51\xe7\x65\x7f\x0f\x4c\xe7\xad\xa8\x8d\xa6\xd8\x57\x8a\xa3\x85\x25\x86\x82\x20\x9b\xc1\x84\x82\xf0\x38\x8a\xb0\xae\xb6\x31\x65\x3c\x44\x52\x9f\x5e\x21\x98\x91\x2b\x4c\x97\xe8\xa7\xf4\xa3\xc9\xd4\xfb\x3e\x6a\x29\x45\xc9\xa2\x01\xd9\xda\x49\xe2\x1b\x99\x90\xb2\x4c\x2a\x55\xd3\xcd\x28\x40\xa4\xb5\x3c\x9a\xe0\x46\x21\x39\xa1\x33\x5b\xb1\xfc\xc7\x9c\x15\xbd\x23\x42\x10\x3a\x3b\x4a\x05\xf1\x0e\xe7\x45\x35\xa0\x1e\x15\xf2\xaa\x67\x46\xdb\xf5\x4c\xca\xe7\xa7\x3e\x50\xd1\xc9\xb1\x0f\x82\x47\x5f\x35\x98\xe6\x71\xd1\xfa\xb6\x8b\xd6\x5a\xfe\x4a\xf5\x4c\x68\x31\x40\xde\x6b\x23\xf0\x18\x4f\x31\xc7\xd4\xcb\xd0\x34\x8a\xd2\x58\x88\xe9\xe7\xb9\x5a\x3c\x24\xb1\xe9\x24\xbe\x4d\x97\x53\xbb\x5e\x10\xba\xbc\xd1\x5c\x11\xd1\xd4\x48\x99\x82\xa3\x6c\xed\x49\x22\x9f\x2d\x2e\xa8\xaf\x58\x3f\x23\x34\xc3\xd6\x4f\x41\xbe\xd8\x3f\x25\x93\x59\xfe\x80\x0e\xba\x97\x38\x14\xab\x11\xde\x8a\x2a\x85\x45\xb5\x91\x72\x6d\x66\x56\x18\xbe\x42\x6e\x79\x2b\x8d\x73\x73\x33\x2d\x9b\x69\x13\x14\x04\xef\xa7\xcb\xe4\x24\x95\xea\x92\x10\xd8\x66\x8e\x83\x1f\x75\x3c\x01\x3d\x11\xfd\x8a\xa8\x3b\x79\x03\x7a\x20\x91\x63\x5a\xd6\x36\xcf\x6c\x97\x71\x51\xec\x9c\x9d\xb2\x12\x34\xb7\x62\x48\xd1\xf3\x58\x99\x0b\x5a\xa0\xdc\x28\x6a\x87\xac\xf4\xc6\xd9\xbc\xb5\x1e\x3a\x4e\x42\xdf\x6c\x62\x5b\x4a\xb1\x17\xc5\x4b\x85\x38\xc4\x21\xe3\x8b\xc6\x66\x29\x06\xc7\x66\x62\x86\xa9\xa2\x59\x01\x93\x34\xa8\xc9\xfe\x4e\x13\xfd\x36\xd1\xf9\x40\x05\x24\x24\x77\x81\x51\x0c\x2c\xfa\x4e\x53\xa9\xaa\xce\x4c\xf3\x30\x0e\x24\x19\xa3\x2f\x2d\xe4\xcd\x2e\x68\x94\xff\x15\x56\xd1\xce\x1f\x28\x88\xb1\x18\xc1\xdf\x28\x89\x47\xde\x80\x88\xe3\x08\xa9\x51\xdc\x30\x47\x44\x82\x30\xaa\x7f\x71\x8c\xfc\xc5\x06\x4c\x75\xbd\x8f\x0d\xf0\x71\xf6\x7a\xc3\xe4\x9f\x13\x3a\xfb\x17\x94\x97\xc1\xda\x89\x5b\x3c\xa3\x6b\x46\xf3\x50\xd7\x98\x99\x82\x3e\x2f\x82\x58\x98\xd8\x12\x1f\x47\x01\x5b\xf4\xe0\x15\xe3\xe9\x6a\x0b\x7b\x9f\x4e\x5a\x63\x90\xf2\xd2\x3d\x27\xab\xc9\x4e\x90\x1c\x95\xb5\x61\x69\x56\xfa\xd0\x3a\x36\x4c\x12\x17\xbd\xd2\x01\x5c\x81\x80\x11\xc4\xa2\x8b\x91\x90\xdd\x81\xf6\xd9\x56\xa1\x47\xa7\xab\xb5\x56\x9c\x3a\x81\xac\x6d\xe3\xb9\x95\x7d\xb7\xb4\x71\x12\x2d\x3e\x46\x95\x2e\xc6\x83\x1b\x81\x8f\x24\xee\x4a\x12\xe2\xb6\x20\xe3\xc8\xff\xda\x20\x8d\x1c\x8f\x57\x5c\x6e\xd2\xba\x8f\x6d\xdb\x17\x0e\x6f\xda\x73\xd0\xa8\x9c\x72\xfb\x16\x41\x98\x27\x11\xf6\x8a\x02\x23\x3c\x9d\xe0\xb7\x0a\xac\x13\xd3\xa5\x0c\x0b\xbb\x8c\xd6\xdb\xa6\x58\x55\xc3\x35\xbf\xe5\x9a\xec\x44\x5d\x9b\x6b\xd0\x59\x16\xbe\xaa\x0d\x36\xe8\x0c\x4a\x6c\x55\x06\x5a\xe5\xa9\x31\xc8\x2a\x8f\xd5\x5a\xde\x66\x00\xda\xe7\xa6\x7d\x5b\x23\xa3\x34\x04\xf9\x5f\xf3\x60\xb8\x72\xfb\x2c\xb9\x5c\x79\xf5\x4f\x2a\xaa\xad\xba\x74\xdb\x16\x87\x41\xc2\x12\xe8\x15\x91\x40\x14\x05\x8b\x2f\xc5\xe9\xb3\xa2\xdd\xe7\xa4\xe3\xf6\xb4\xa4\x7f\xc2\x43\x01\xa1\x33\xf7\x58\x57\x90\x6b\x42\x50\xd3\x19\x4b\x76\xe2\x86\x08\x4d\x9a\xca\x10\xa8\x5d\x6a\x07\xd4\x3a\xbf\x25\xff\x4b\xd5\x37\xa1\x72\x6b\xe8\x78\x1f\x12\x4a\xc2\x38\x1c\xc1\xa0\xf2\x32\x24\xf4\xf8\x07\x7d\x19\xdd\x7c\xe7\x2f\xfb\x93\xfb\x23\x80\xa5\x4a\xb4\xdf\xc9\x20\x6e\xd2\xe0\x7b\x47\x07\x09\x52\x25\xb5\xab\x5e\x5e\x95\x74\xf1\xdc\xa0\x55\xd8\x7f\x2f\xb6\xf0\x58\x10\x60\x9d\x2f\x5f\xe1\x58\xd7\xc0\x4c\xb7\xeb\xab\xd9\x82\x2e\xe8\x9b\xf5\xcd\x8b\x4b\x50\x79\xed\xa9\xf7\xa7\x6b\x11\xfc\x5e\xaa\xde\x39\x80\x85\xd2\xa3\x29\xcc\x62\x4d\x68\xdd\x3d\xcb\xa5\x49\xd3\x12\x93\x32\x9a\xe9\xf9\x01\x4c\x98\x9f\xa2\x5f\x19\xf7\x62\x59\x08\xf3\x17\xa2\x9b\x71\x5a\x55\x73\x9c\xa4\x23\x16\x52\xe0\xdb\xba\xbe\x0e\xd8\x95\x94\x3e\x5d\xb9\x3b\x4f\xe7\x43\x11\x49\xf0\x5f\x3e\x43\x6b\xbd\x3a\x07\xfe\x2d\xe4\xc0\x49\x76\x93\x73\x72\x40\x7d\x9d\xcd\x74\xfb\xda\xa4\x45\x02\x6d\x1f\xc9\x59\x5b\xa1\x8d\x10\x64\x55\x46\x98\xbf\x00\x81\xa9\x54\xbe\x5d\x36\x67\xe0\xe8\xfd\xc9\xe9\x5a\x1d\xfb\xba\xda\x8d\x59\x71\x90\x6b\x1d\xcf\xca\x50\x97\x22\x25\xaf\x75\xa1\x7a\x3b\x23\x2e\x9b\x19\x99\xbf\x97\xa6\xca\x12\x5a\x11\x87\xf2\x1e\x8a\xc3\xfd\x2c\xf2\x09\x4b\x93\xc7\x28\x99\x96\x65\xf5\xaf\xc7\xe8\x94\xcc\xe2\x5a\x34\x24\x53\x48\x68\xd0\x7b\x7f\x55\x30\x28\xfb\xb4\x65\x1f\xb0\xe8\xc9\x2a\x0e\xd0\xc4\xf3\x76\x7e\xad\x07\x07\x12\xc2\x58\x48\x85\x96\x48\xe2\x4c\x03\x76\x8d\x79\xd7\x43\x02\x03\x0a\xa2\x39\xa2\x71\x88\xb9\x72\x7a\xe7\x88\x23\x4f\x62\x2e\x80\x71\x58\xef\xae\x6f\xa8\x69\xc3\x93\xa2\x3e\x88\x9a\xd6\x13\x2c\xed\xb6\xa6\x18\x1c\xa6\x7e\xb1\x55\x05\xa6\x69\xe7\x21\xaa\x4f\x4c\x26\x18\x02\x46\x67\x58\xc9\x17\xa2\xb0\x35\xb4\x3e\xde\xeb\x2c\x1b\x97\xaa\x93\xef\xc8\xea\x55\x4d\x1a\xe5\xc1\x4b\x2a\xfe\xaf\x2e\x17\x0e\xe7\x6f\x65\xc7\xcf\xe1\xf4\xad\xe4\xf0\x55\x32\x04\x7e\x98\x73\x56\x9f\xaa\x00\x3f\xca\x37\xb3\x51\x7a\x30\xae\x99\x8d\xb4\x35\xc6\x79\xf0\xf5\x0f\x1d\x61\x57\x0c\x38\xfc\xc8\xf1\xad\xad\xda\x73\x7f\x47\xd7\xa0\xec\x98\xbf\xce\x45\xd8\xf0\x3d\xdf\x84\xec\x34\xac\x98\xe5\x73\xad\x22\xa0\xdc\xa8\x50\x9a\x50\x91\x9c\x25\xe0\x1b\x41\xe8\xc1\xa7\x44\x09\xae\x17\xf0\x5a\x87\x80\xd0\x8b\xe5\x0a\x99\x34\x7c\xfc\x23\x25\x97\x31\x06\xe2\x63\x2a\xc9\x94\xe0\xbc\x18\xa9\xf9\xf4\x52\xe0\x3e\x11\x51\x80\x16\xe3\xe6\xa5\xf0\xd0\x5a\x06\x4b\x46\x81\x32\x63\x12\x20\x10\xc5\x3c\x62\x02\xb7\x58\x64\x9a\x3f\xf7\x26\x0e\x11\x85\x29\x27\x98\xfa\xc1\xc2\x41\x5d\x11\x87\x0d\x8d\x44\xba\x05\x7e\x86\xae\xc5\xd9\x72\x0c\x30\x45\x93\x00\x37\xb0\xf6\x53\x62\x22\x3a\x68\x26\x22\xed\x6e\xc8\xd7\x1b\xf1\x84\xce\xd4\x02\xfd\xfe\xe4\x65\xba\xf8\x39\x90\x28\x5a\x20\x2e\x83\x2e\x01\x5c\x56\x51\x6e\x21\x7e\x99\xff\x52\xac\x41\xe9\xca\xac\xff\xdf\xfb\x61\x12\x6e\x50\x5e\x7f\x70\xa2\x9d\x70\xcf\x25\xd2\x25\x19\x3b\xec\xc1\x1f\x84\xcf\x08\x25\xe8\x6b\xcb\x5a\x82\xc4\xd7\x92\x31\xf3\xb1\x29\x8a\x03\x39\x82\x29\x0a\x44\x7e\x42\x90\xa5\x44\x8d\x0b\xdb\xf4\xf5\xee\x9f\x36\x89\xab\xb6\x5e\xda\x5b\x7f\x5f\x58\x99\x56\x69\xbd\x1c\x43\x92\x03\xd5\xf2\xd2\xe0\x58\x16\x1c\x0c\x5d\x36\x6d\xcc\x93\x3a\xea\x74\x93\x9f\x0a\x19\x23\x69\xd8\x5d\x9a\x39\xf2\x93\x6e\xe3\x4c\x37\xa8\x35\x13\x6e\xb1\xf4\x2f\xc9\x67\x80\x74\x0b\x67\x40\x27\xd1\xc9\x93\xfe\x1b\x3f\x3e\xc2\xdb\x41\x5f\xb2\xa7\xe7\x27\xb3\xe1\x8b\xb7\x5f\xa6\x71\x75\xad\x6f\xb5\xcb\x95\x2d\xf6\x15\x14\x56\x58\xf2\xcb\x4a\xa3\x66\xb4\x32\x42\x5a\x37\xfd\x91\xa6\x44\xce\x89\x24\x90\x20\xfb\xbd\xe2\xce\xb9\x91\xa9\x16\xdb\x96\xc8\xf7\x89\x9a\x5e\x28\x38\xaa\x61\xb4\x93\x53\x57\xe6\xe8\xfa\x36\x53\xaa\x89\x7e\x03\xd6\xd0\x5e\xfc\x44\x4b\xba\x75\x1e\x94\x44\x61\x54\x45\xad\xba\x25\x6c\x6d\x05\xef\x6e\x17\x49\xab\x76\x37\x55\xba\x1c\xbd\x7d\x16\x4f\x02\xdc\xa0\x1c\x34\x40\x7b\x4e\x97\x03\xea\xbf\xc1\xac\xae\x8f\xd9\x87\xef\x37\xaf\x6d\x24\xfe\xdb\x67\xb6\xcd\x8b\x8e\x2d\x0c\xaf\x4c\xc4\x37\x61\xf4\x18\x0b\xb5\x4c\xae\xd5\x90\x61\x43\xb8\x67\xda\xe0\xfe\xce\xba\xb5\x6a\x8c\x66\x2e\x31\xa6\x8a\x2f\xa9\xb9\x8a\x09\xc3\xc1\x4b\x73\xb7\x94\xc7\x78\x56\xaf\xb9\x14\xa0\xea\x60\x07\xa1\x23\x88\x90\x9c\x97\x71\xcb\x77\xc4\xd3\xe4\xab\x22\x1e\xe9\x53\x0b\x8c\x5d\x76\xb8\x82\x5d\x80\xe9\x4c\xce\xb5\xf1\x48\x42\x0c\x84\x42\x48\x68\xac\xec\x64\x65\xb1\x5d\xcf\x89\x37\x07\xc9\x92\x6a\x5c\xa6\xe0\x60\x21\xa9\xc0\x59\xad\xd6\x4d\x5f\x79\x10\xdd\x43\x98\xd9\x79\x3b\xb9\xe0\x55\x0f\xd3\x92\x3d\xef\x11\x6c\x6f\x0d\xfb\xc9\xd3\x6a\xb4\x6e\x99\x45\xb9\x88\x24\xd0\xd3\x6c\xb4\xd2\x58\x26\x4f\xdb\xf2\x30\x6d\xaf\xd3\x36\xb0\xc7\xa8\x2f\x60\x82\xe5\xb5\x2e\x70\x87\x24\x82\x2c\x85\xf7\xdb\x72\x6c\xab\xdf\x8a\x65\x83\xfe\xd3\x7e\x3d\xcf\xca\x2c\xb1\x78\x96\xc0\x4f\x32\x60\x8a\x3c\x4b\x1e\xb6\x61\x59\x7a\x45\x60\x6a\xb1\x4a\x06\x53\x2c\xbd\x79\x0f\x5e\xa9\x7f\x0a\x89\x30\xd7\x73\x4c\x01\x87\x91\x5c\xf4\x4c\x3f\x65\xb8\x13\x2c\x74\xd2\x74\x6a\xc5\x6b\x94\x69\x96\x7a\xa2\x31\x12\xbd\x46\xce\x16\x55\x75\x45\x49\xbb\x7d\xb1\x94\xcf\x59\x8d\xf4\x3c\x0a\xd8\x30\xc1\x8a\x4e\x6e\xe4\xc0\x91\xb9\x08\xd6\xc7\x37\x15\x99\xb0\xbd\x9b\x16\x6a\xa2\x3a\x7e\xe5\xd8\xe4\x64\xec\xd2\x2d\x35\x3b\x28\xd9\x20\x6d\xc5\x50\x37\x22\x7d\x98\x17\x18\x55\xec\x52\xc2\x8e\x91\x37\xb7\x89\xfe\x8a\x64\x94\x83\xa7\x33\x32\xfa\x7d\x43\x48\x52\xc0\xc8\xb9\xa7\xf0\xef\x6e\xd6\xf3\x24\xb9\x69\x20\xb9\xcf\x4e\x75\x52\x2e\x9d\xc7\x89\xc4\x9c\xa0\x9e\x9e\xc0\x62\x41\x25\xba\xc9\x36\x04\x32\x5d\x0f\x44\x58\x08\x85\x24\x40\x3c\xbd\xae\xd8\xee\x82\xe1\x2c\x05\x7c\x06\x5e\xa0\x4b\xb3\xb2\xa9\x72\x72\x4f\x3e\xbc\xd5\x07\x8b\xd8\x5c\xb4\x9c\xc2\xd2\xe5\xec\x4d\xae\x69\x52\x9d\x55\xf7\x37\xde\x26\xa2\x8b\x14\xec\x94\x05\x01\xbb\x56\xde\xdd\x99\x57\x38\x9a\x13\x67\x30\x25\x38\xf0\xc5\x68\x2d\x03\xfa\x0b\x24\x37\xed\xa6\x3f\x8b\x87\x64\x85\x17\xda\xed\x1e\x5b\x69\x3b\xbf\x58\x11\x7f\x36\x44\x8e\xa7\xd6\xcf\x42\x87\x82\x47\x6a\x3d\xaf\x84\xb7\xfe\x62\x1f\xf0\xa9\x9f\x76\xb5\xc6\x22\x12\x3a\xfc\xd1\xfa\x6d\x9c\x6e\xeb\x41\xe9\x94\xf6\x17\x2b\xaa\xd0\x7a\x98\x44\xf8\xe5\xbc\xb1\xa2\x33\x37\xac\xe5\x4c\x69\x9a\x5c\x89\x64\xf7\x32\xe7\x63\x21\xe7\x98\x70\x8d\xfe\x06\xa4\x05\x77\xf3\x41\x31\x32\x60\x0d\xc1\xd9\xd9\x99\xb8\xcc\x13\x10\xf4\xc1\x1b\x12\x9e\xfd\x3e\x6f\x7c\x7a\x1b\x34\x60\x8c\xa8\x3f\xce\x4e\xa2\x14\xed\x77\xc1\x6c\xc3\x1a\xf6\x7a\x4c\xcd\xfd\xde\x85\x69\x41\xd7\x65\xba\x2b\xe7\x6f\x00\xe3\x40\x4c\x1b\x3d\x4d\xf5\xbe\x8f\xd2\xd9\x1b\xea\x59\x3e\x7c\x66\x67\x48\x99\xa7\x46\x7f\x5b\x14\x2a\x84\x7a\x99\x32\x88\x02\xe6\xe3\xc2\xfa\x58\x55\x10\xa5\xf9\x0f\x96\x8e\x48\xa9\xeb\xd4\xa8\x35\xa3\xf7\x12\x00\x77\x55\x5d\x42\x2e\x02\xb5\xfe\x31\x6e\xb2\xab\x4d\x99\x34\xb7\x5a\xca\xb5\x92\x6e\x94\x6b\x21\x4b\x2a\x9a\xd5\xd1\x12\x35\xa4\x4f\x33\x8b\x3a\x28\xff\x66\x41\x17\x41\x52\x42\x3b\xd1\x23\xd9\x4d\xa0\x06\x31\x35\x3a\x67\x45\xfd\x71\xb6\x01\x67\x8a\x71\xea\x5f\x3d\x4d\xd5\xff\x98\xf9\x79\x66\x8e\x6e\xcf\xcc\xe4\x3c\xcb\x61\x2b\x0f\x06\xe9\xfb\xc5\xcc\x80\x9f\xfd\x9f\xff\xab\x7a\xfd\x7a\xa6\x45\xe6\xec\xed\xc1\xef\xfb\x67\xb9\x56\x4c\x7b\x9d\x33\x42\x93\xf6\x7b\x87\x2f\xcf\x0c\xec\xf7\xc7\x67\x3d\x78\xc3\xae\xf1\x15\xe6\x1b\xb0\x60\xb1\xd6\x9c\x8a\x4a\x94\x05\x40\xb0\x29\x0c\xfa\x49\x77\x9d\x7d\x9a\x50\xa3\xc7\xde\xe2\xf1\x7e\x26\x4c\xae\xc9\x58\x99\x8a\x49\x6d\x94\xf4\x24\xfd\x2c\x5c\x74\x13\x6d\x6c\x70\xb3\x36\x3c\xf5\xbe\x7d\xdb\x09\x59\x9c\x8d\xbf\x42\x0e\xd7\x9c\x84\x17\xd8\x0f\xbf\x02\xba\x16\x76\xf7\xbf\xa3\xee\xbf\x56\x21\x00\x99\xef\xe8\x02\xba\xfa\xd4\x3e\xa9\x7d\x70\x16\x2e\x6e\x89\x72\x40\x2e\x30\x84\x8b\x7f\x0c\x77\xbe\x89\xde\xd0\x7a\xd1\x3e\x99\xcf\x74\xa3\xa5\x51\x90\xcc\x8b\xfd\xce\x91\x80\x08\xf3\x90\x08\x7d\x25\xba\x64\x20\xb0\x29\xaf\xc3\x93\x24\x65\x4b\x08\x0e\x99\xc4\xbd\x14\x45\xb3\x16\xe7\xe9\xac\x4a\xa0\x93\xa4\x44\xbd\xf5\x9b\xf6\xae\x57\x50\x89\x2d\xa5\x05\xae\x46\xed\xb8\x55\x8c\xc3\xf4\x29\x68\x10\x28\x2b\xb6\x56\x82\xd2\xb9\xbd\x02\x73\x86\xbb\xa7\x8e\x51\x75\xc9\xaf\x78\x43\x15\xe9\xd3\x87\x2d\xca\xc8\xd7\x2e\x42\x61\x15\x98\x2c\x6a\x78\xd5\x02\xef\xb6\xec\xc4\x57\x28\x18\xd7\xc6\xf0\xa7\xac\xc5\x85\x9a\x24\xaa\xb1\x8f\xb8\xbf\xbc\x5f\xda\x52\xf5\x4d\x13\xec\x75\x70\x54\x8a\x42\x92\x61\x6f\xd3\x85\x47\x30\xd1\x4f\x93\x87\xe6\xc7\xab\xc4\xb9\xfb\xed\x53\x1a\x2b\x65\x88\x9e\x4b\x19\xad\x95\x09\x2b\xdd\x49\x93\x82\x2f\xed\x7f\x24\x61\x30\xd0\xc9\x12\x52\x72\x12\x4b\x21\x54\xd0\xb1\xa4\x26\x1d\xef\x4e\x12\x52\x86\x22\x22\xb3\x68\xf0\xd2\xc5\x35\xcb\x3e\x8d\xe3\xee\x35\xfe\x4a\x9f\x76\x86\xd4\xd7\x20\x60\x76\x27\xc9\xc9\xe7\xdd\xe3\x0f\x5b\xbf\xfd\x7e\xf0\xf4\x43\xff\xfd\x69\x78\xfe\xe1\x95\xbf\xc5\xbc\x57\xc7\xb3\xfc\x83\xc9\x9e\x67\x32\xa5\xf2\xe7\x8d\x71\xa0\x9b\x4d\xa0\xe1\x27\x88\x38\x9a\x85\x68\xa4\x74\x18\xbb\xd6\xf5\xfb\x04\xf6\x38\x96\xb6\x88\xc9\x58\x8c\xa0\xa3\x73\xb2\xda\x32\x27\x0b\x38\xb3\x75\x4f\xf3\x40\x9b\x94\x22\xe8\xa0\x88\x8c\x13\xec\xc7\x09\x73\x1b\x98\x6e\xb1\x81\x09\x59\x78\xd5\x1d\x10\xb1\xd8\xe5\x97\x5b\xe7\x17\xe4\xe9\x65\x9f\xc9\xf0\xfc\x72\xaa\x48\x9f\xf2\x59\x2f\x11\xd3\x9e\x66\x58\xcf\x63\xa1\x45\x59\x9e\x53\xa4\x2f\x2f\xea\x77\x07\xfd\x6e\x7f\xe7\x74\x30\x1c\xed\x0c\x46\xc3\xed\x5e\x7f\x67\x6b\xb0\x3d\xfc\x2b\xef\x61\xa5\x0c\x55\x7a\xec\x8e\xb6\x76\x7b\x5b\xbb\xc3\x61\xff\xa9\xd5\x23\xcd\xed\x81\xce\xb0\xb7\xdb\xeb\xe7\x2f\x8a\x13\x39\x9b\xe0\x0e\xb1\x6a\x71\xcd\xf6\x43\x15\x35\x93\x2f\xf5\x28\x6b\x5f\x47\xd6\x8a\xd9\x67\xd0\x41\x49\xde\xb3\xeb\xbe\x85\xfc\x62\x8b\xf2\x18\xb4\x17\x4c\x47\x65\xfd\x92\x20\xb6\xc8\x86\xaa\xc6\x5d\x55\xe3\xb3\x1c\x51\x58\x95\x1d\xb2\xee\x0a\x52\xdf\x24\xf9\xdf\x5e\xfa\xa1\x41\xd9\x2e\x9f\x04\x0d\x13\x61\xd9\x64\x00\x7b\x42\x04\x85\x29\x00\xcd\xd3\x00\xbe\xe6\x54\x80\x5b\x4d\x07\xb8\xd5\x94\x80\x26\x15\x0c\xcb\xa4\xdd\x11\x94\xb9\x44\xd0\xab\x11\x91\x90\xea\x63\x97\x4d\x51\x78\x56\x08\xad\x81\xce\x5e\x88\xbe\x30\x0a\x9f\xf0\x24\xcd\x81\xb0\xda\xa6\xc1\x2f\xb9\x00\x54\x23\x0c\x5b\xa0\x6a\x87\xf7\x65\x88\x3a\x24\xa7\x84\xda\xc7\x13\xd8\x47\x42\x6e\x80\x15\xb1\xd3\x84\x1b\x34\xc5\xc5\xc0\xdf\xb9\xcd\xba\x91\xd8\xbd\x66\x3f\xbb\x2e\x8e\xa2\x86\xb0\xea\x71\xe0\x58\x23\x3c\x1e\x8f\xc0\xd6\xfd\x98\x8f\x27\x9c\x5d\x60\x2e\x59\x44\xbc\x64\x63\x7f\x3c\x59\x48\x2c\xc6\x84\x8e\x8b\x35\x3b\xb2\xd0\xe9\xb1\x29\x0a\xc1\xf8\x98\xb0\x71\xb2\x61\x99\xc1\xed\x96\xef\x21\x50\x3a\x2b\x22\xde\x08\xc6\x63\x8f\x51\x11\x87\x98\x8f\xd9\x74\x2a\xb0\x75\x9b\x4a\x35\xd8\xa0\x6b\x1d\x39\xc2\x60\x77\x30\xd8\x7d\xd2\x1f\x6e\xf5\xfb\xfd\x7e\x51\xa0\x8d\xa9\xff\x74\x7b\xb0\xb3\xbd\xac\xf7\x6e\x6d\xef\x9d\xa7\x4f\x9f\x2e\xeb\xfd\xac\xb6\xf7\x93\xdd\xe1\xd0\x1e\x24\xc7\xa1\xf8\x7f\xc8\x30\x2d\x1d\x92\xca\x70\xd4\x5f\x33\xee\xb4\xd3\xfa\x5b\x15\x7b\xac\x54\xfc\xc7\xb9\x1e\x99\x9b\x10\x37\x0b\xdd\x75\x51\x14\xe8\xe8\x6c\xa2\xee\xbb\xd7\xef\x4e\xbb\x85\xd7\x99\x65\x70\x62\xdd\xd5\x9a\xde\xe2\x6a\x6e\x7b\xcb\xa6\xa9\xd9\x81\xd1\xb7\xba\xfe\xaa\x13\x45\xb2\x5d\x13\xcb\xba\xb2\xeb\xa4\xa8\xb5\xf7\xd3\x01\x09\x2f\x5f\x7b\xfc\x65\xfc\x76\x77\x80\x3e\xde\x1c\xfc\x75\xf9\xfc\xf4\xf2\xf0\x38\xd1\x0e\xf5\xb7\x46\x3e\x32\xa6\xe1\xee\x1d\x17\x73\x86\x77\xe2\xcd\xb0\x91\x35\x43\x17\x67\x8c\x17\x02\x92\x29\x7a\x05\x2e\x6c\x65\x8e\xe0\xa3\xd6\xf9\xea\xad\xb6\x78\xca\x37\x48\x99\x52\x81\x55\x1b\x7c\x04\x85\xcf\x8e\x60\xd9\x57\xac\x82\xf3\x2c\x88\x43\x6a\xf6\xfd\x14\xf4\x64\x93\x0a\xd6\x89\xbf\xde\x83\x13\x57\x3b\xbd\x87\x3b\x4a\x4c\xa4\x8d\xe4\x14\xa5\x68\x6d\xa5\x4f\x8d\x7d\xd6\x83\x0f\x66\x1f\xce\x8c\xcd\x08\x88\x0f\xbf\xc2\xc0\xe6\x4f\x79\xa4\x83\x4f\x2f\x5f\xc7\x8b\xc9\x01\xdf\xa7\x37\x7c\x0f\x87\x4f\x86\xdb\xb3\xcb\x8b\x0b\xf2\xf2\x2a\x1b\xe9\x25\x95\xf3\x9c\xa3\x3d\xb8\xd3\x68\x0f\x1a\x47\x7b\xe0\x18\xed\xd0\xe0\xa8\x23\x3e\x72\x01\xcf\x14\x29\xd8\x15\x2a\x57\x67\x41\xb9\xa6\x9b\x8b\xe4\x27\x77\xa1\xf8\x49\x13\xc1\x4f\x1c\xf4\xb6\xb9\x56\x31\xc3\xde\x79\xc9\xfa\x8f\xa7\x21\xf3\xb2\x12\xe4\x4d\x21\x61\xff\xd7\xf5\x01\xf9\x7d\xcb\x8f\xff\xf8\x7c\x70\x75\xb5\xf3\xf9\xea\x6d\xb0\xf8\x32\x08\x5f\x1f\x6f\xfd\xb6\xb8\x3c\x5c\xcf\xab\x01\x36\x68\xaf\xcf\xef\x9f\xcc\x86\xb3\xdd\x37\xa7\xfe\xc7\xdf\x3f\xa2\xe1\x85\x78\xf3\x74\x78\xf1\xe1\xe5\xd6\x22\x65\x49\xb9\x90\xa1\x53\x9d\xdf\x49\x86\x07\x8d\x32\x3c\x70\xc9\x70\xae\x93\xae\x30\x27\xd3\x05\xfc\xf6\xe9\xd4\xd4\x89\x1c\x41\x92\x81\xee\x67\x37\x4b\x19\x5f\xd8\x54\x91\x6c\xc5\x92\xad\x8f\xf3\xfd\xf9\x75\xf8\xe7\xf3\xe8\xd3\xd1\xf4\x60\x18\x1c\xe2\x8b\xc8\xdf\xfe\xeb\x65\xca\x92\xf2\xbd\x58\x2e\x96\x6c\xdf\x85\x23\xdb\x4d\x0c\xd9\x76\xf1\x43\x60\x0e\xeb\x53\xc6\xba\x13\xc4\xd7\xdb\x5e\xef\xda\x6b\x98\xe8\x9f\xb7\x3e\x92\xfd\xf9\x17\x6a\x31\xe1\x3c\xf2\xb7\x3f\xbf\xc8\x98\xd0\xf6\xba\x5d\x17\x77\x76\xee\xc2\x9d\x9d\x26\xee\xec\x2c\xe7\xce\x1c\x89\x34\x4b\xd9\x3a\xd0\xcb\xef\xb4\xdd\x35\xfe\xbd\x4e\xbe\x4c\x0e\x81\x96\x72\xea\xe2\x46\x71\xea\x8f\x23\x7c\x30\x64\x87\xf8\xdc\xdf\xfa\xf3\x79\xc6\xa8\x25\xd7\x01\x3b\x27\xd4\xf0\x4e\x13\x6a\xd8\x38\xa1\x86\x0e\x16\x65\x93\x46\x2a\x64\x61\x8e\xae\x70\x92\x1a\x8b\x29\xa4\xc5\xbe\x6a\x99\x70\xf1\xe7\x8b\x2f\x9f\x34\xed\x29\x13\xde\x5e\xbd\x7a\x76\xfe\xee\xc3\xe7\x94\x09\x4d\x77\xc0\xba\x18\xb0\xb5\x7b\x17\x06\xd8\xbd\xab\x0c\xb0\xdf\x16\x34\x2c\x0a\x8c\xcd\x41\x04\xa0\x40\xef\xde\xe8\xda\x61\xb5\x64\xef\x5e\x7c\xee\xab\xb1\xff\x92\xd3\xff\x19\xcf\xfd\xad\xfd\x44\x53\x54\x0b\x77\xba\x48\x7d\x76\x17\x4a\x9f\x35\x11\xfa\xcc\xa9\x39\x93\xb2\x62\x69\x25\xd4\x06\x45\x88\xf7\xd3\x61\xdc\xfd\x3c\x9b\x4f\xdf\x3d\x9b\xbd\x3e\x16\x6f\xae\xf6\x3f\x65\xe4\xb5\x5e\x2e\xbf\x27\x91\x96\x0b\xd8\x31\xe7\xaa\x69\xb9\x39\x50\x96\xbd\xc0\x72\x04\xef\x5f\xbc\xeb\xee\xff\xd9\x7d\x36\x4a\xf6\x48\x4d\x7d\x38\x45\x4c\xde\x06\xdf\xc8\x6e\x61\x4b\xec\xa6\xbf\x15\x50\x3f\x08\x2f\xfb\x97\x53\xef\x89\x20\x12\xed\x88\xe0\xfc\xea\xa9\xed\x74\x2a\x7b\x35\xcd\xe5\x56\x94\x0f\x66\x3b\xfe\xd3\xa7\x97\xfd\x80\x7b\xfe\xd5\xf6\xec\x09\x0a\x26\x4f\x44\x30\x9d\xd1\xf3\x2d\x7f\x3e\x11\xe7\xff\xf8\x5f\xff\xdc\xff\xf3\xf4\x78\x0f\x7e\x31\xc4\xf6\x34\x5f\x7e\xcd\xb3\xb8\x2c\xd8\x44\xc0\xfa\x76\x7f\x7b\x7d\x43\xb3\x41\xff\x7c\xf1\xf6\xe3\xc9\xe9\xfe\x71\xba\x34\xf4\xb7\xd7\xf5\x31\x6d\x36\x94\x76\x3a\x98\x6a\x3f\x98\xed\x30\xbe\xd3\xbf\x22\x71\xff\x09\xc3\x6a\xa0\xe6\xfc\xc2\x1b\xee\xfa\xb3\xa9\x3c\x1f\x20\x6f\xdd\xe6\xde\x8b\x84\x8e\xf5\x65\x44\x58\xa6\xc6\xcf\x4d\x0b\xeb\xa9\xf8\xc4\x17\xbb\x54\x5c\x4e\x86\xe2\x30\x7c\x75\xbe\x33\xf9\x33\x7a\xf9\xe4\x05\xea\xac\xfd\x4f\x00\x00\x00\xff\xff\x64\x4f\xfa\x91\x78\xa0\x00\x00")

func fleetManagerYamlBytes() ([]byte, error) {
	return bindataRead(
		_fleetManagerYaml,
		"fleet-manager.yaml",
	)
}

func fleetManagerYaml() (*asset, error) {
	bytes, err := fleetManagerYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "fleet-manager.yaml", size: 41080, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"fleet-manager.yaml": fleetManagerYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"fleet-manager.yaml": &bintree{fleetManagerYaml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
