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

var _fleetManagerYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5d\xeb\x57\x1b\xb9\x92\xff\xce\x5f\x51\xeb\xd9\x3d\xbe\xb3\x07\x1b\xbf\x78\xc4\xe7\xcc\x07\x02\x24\x61\x6e\x20\x09\x8f\x21\x99\x7b\xee\xf1\x95\xbb\x65\x5b\xd0\x2d\x35\x92\xda\x60\x76\xf7\x7f\xdf\x23\xa9\xdf\x6f\x1e\x49\xe0\x0e\x7c\x49\xe8\x96\xd4\x55\x3f\x95\xaa\x4a\x55\x25\xc1\x3c\x4c\x91\x47\xc6\x30\xec\xf6\xba\x3d\xf8\x05\x28\xc6\x36\xc8\x05\x11\x80\x04\xcc\x08\x17\x12\x1c\x42\x31\x48\x06\xc8\x71\xd8\x0d\x08\xe6\x62\x38\xdc\x3f\x10\xea\xd1\x15\x65\x37\xa6\xb5\xea\x40\x21\x18\x0e\x6c\x66\xf9\x2e\xa6\xb2\xbb\xf6\x0b\xec\x3a\x0e\x60\x6a\x7b\x8c\x50\x29\xc0\xc6\x33\x42\xb1\x0d\x0b\xcc\x31\xdc\x10\xc7\x81\x29\x06\x9b\x08\x8b\x2d\x31\x47\x53\x07\xc3\x74\xa5\xbe\x04\xbe\xc0\x5c\x74\xe1\x70\x06\x52\xb7\x55\x1f\x08\xa8\x63\x70\x85\xb1\x67\x28\x89\x47\x6e\x79\x9c\x2c\x91\xc4\xad\x75\x40\xb6\xe2\x01\xbb\xaa\xa9\x5c\x60\x68\xb9\x88\xa2\x39\xb6\x3b\x02\xf3\x25\xb1\xb0\xe8\x20\x8f\x74\x82\xf6\xdd\x15\x72\x9d\x16\xcc\x88\x83\xd7\x08\x9d\xb1\xf1\x1a\x80\x24\xd2\xc1\x63\x38\xc1\x36\x7c\x40\x12\x76\xed\x25\xa2\x16\xb6\x61\xcf\xf1\x85\xc4\x1c\x4e\xb1\xe5\x73\x22\x57\x70\x6a\x06\x84\x77\x0e\xc6\x12\x8e\xf4\x67\xf8\x1a\xc0\x12\x73\x41\x18\x1d\x43\xbf\x3b\xe8\xf6\xd6\x00\x6c\x2c\x2c\x4e\x3c\xa9\x1f\xd6\x8f\xfb\xb7\x93\x0f\xbb\x7b\xa7\xbf\x16\x8f\x6f\xb0\x38\xc1\x42\xc2\xee\xe7\x43\xc5\xa4\xe1\x0f\x08\x15\x52\x0d\x28\x80\xcd\x60\x77\xef\x14\x2c\xe6\x7a\x8c\x62\x2a\x45\x77\x4d\xf1\x8e\xb9\x50\xec\x75\xc0\xe7\xce\x18\x16\x52\x7a\x62\xbc\xb1\x81\x3c\xd2\x55\x33\x27\x16\x64\x26\xbb\x16\x73\xd7\x00\x32\x14\x1f\x21\x42\xe1\x6f\x1e\x67\xb6\x6f\xa9\x27\xbf\x82\x19\xae\x78\x30\x21\xd1\x1c\xd7\x0d\x79\x2a\xd1\x9c\xd0\x79\xe1\x40\xe3\x8d\x0d\x87\x59\xc8\x59\x30\x21\xc7\x3b\xbd\x5e\x2f\xdf\x3d\x7a\x1f\xf7\xdc\xc8\xb7\xb2\x7c\xce\x31\x95\x60\x33\x17\x11\xba\xe6\x21\xb9\xd0\x08\x28\x32\x37\xf8\x02\x59\x62\x63\xd9\x1f\xeb\x7e\x73\x2c\xcd\x7f\x40\x89\x31\x47\x6a\x80\x43\x7b\xac\x9e\xff\x61\x66\xf3\x08\x4b\x64\x23\x89\x82\x56\x1c\x0b\x8f\x51\x81\x45\xd8\x0d\xa0\x35\xe8\xf5\x5a\xf1\xaf\x00\x16\xa3\x12\x53\x99\x7c\x04\x80\x3c\xcf\x21\x96\xfe\xc0\xc6\xa5\x60\x34\xfd\x16\x40\x58\x0b\xec\xa2\xec\x53\x80\xff\xe4\x78\x36\x86\xf6\x2f\x1b\xf1\xb4\x6e\x98\xb6\x62\x23\x43\x62\x3b\xd1\x39\x05\x48\xd0\x0e\xdc\x34\x2f\xc2\x77\x5d\xc4\x57\x4a\x34\xa5\xcf\xa9\xd0\xcb\x66\x99\x6d\x9b\x05\x6e\x03\x73\xce\xb8\xd8\xf8\x1f\x62\xff\x5f\x2d\x88\x07\xaa\xed\xdb\xd5\xa1\xfd\x1c\xe1\xd3\xc4\x95\x82\xf6\x1e\x4b\xd0\xac\x2a\xe5\x14\x31\x50\x88\x59\xd4\x8c\x84\xcd\x24\x9a\x27\x58\xec\x98\x16\x22\x78\xe0\x21\x8e\x5c\x2c\x83\x75\x19\x36\x31\x94\xb6\x52\x94\xc6\x2d\x37\x88\xdd\x2a\x9b\x8a\x66\xb3\x20\x9e\xed\x14\x7c\x24\x42\x96\x4e\x83\x7a\xa9\x34\x9b\xc7\x84\x20\xca\x54\xa4\xa0\x2c\x9c\x0e\x27\xdb\x45\x29\xcc\x54\xb7\x92\xe9\xc9\xe1\x2b\x24\x92\x7e\x3d\xbe\x81\xc2\x3e\xd5\xad\x9f\x23\xcc\x29\x02\x4b\xa1\xfe\x74\x15\x93\xba\x99\x21\x35\xd5\xf0\x9c\xe2\x5b\x0f\x5b\x12\xdb\x81\xe8\x33\x4b\xeb\x5c\xfb\x59\xac\x62\xf5\x83\x6f\x91\xeb\x39\x49\xf0\xc3\x9f\xcd\x5e\xef\xc0\xbc\xcc\xbf\x2b\xfe\x50\x38\xd6\x46\xdc\xb5\x5d\x25\x7e\x46\x68\x94\x00\x72\x2c\x98\xcf\x2d\x2c\xd6\x41\xf8\xd6\x42\x79\x57\x37\x0b\xac\x5c\x1b\x70\xd1\x2d\x71\x7d\x17\x02\xe7\x04\x2c\xe4\x21\x4b\x39\x01\x0b\x24\x60\x8a\x31\x05\x8e\x91\xb5\x88\x20\x15\x81\x93\x90\x94\xda\xb7\x18\x71\xcc\xc7\xf0\x0f\xf8\x67\x4e\x72\x2d\x4c\x25\x47\x4e\x43\x35\xbd\x67\x5a\x27\x14\x75\x6a\xbe\xcf\x94\xb3\x17\xf5\x51\x9e\x08\xa3\xce\x0a\x90\x2f\x17\x8c\x93\x3b\xe3\x9e\x69\xdf\x0d\x08\x35\x18\x20\x17\x03\xe3\x73\x44\x89\x30\x9d\x90\x01\x87\xdd\x50\xcc\xd3\x6f\xd8\xcc\x74\xf1\xb0\x45\x66\x44\x39\x46\x86\x9a\xee\x73\x5c\x49\x01\x6d\x27\xf8\xda\xc7\x69\xad\x55\x2d\x76\xe9\x7e\xef\xb1\x3c\x09\xb8\x7a\xa8\x30\xa6\x07\xcc\xc8\x65\x83\xef\x5e\x10\xb9\x78\x87\x88\x83\xed\x3d\x8e\x35\x46\x46\x3b\x3c\x0d\x3d\x15\x23\x97\xaa\x9f\x60\x04\xe0\x66\x08\x98\x31\x9f\xda\xda\xf8\xee\xc7\x13\x3f\xea\xf5\x9f\x89\xb3\x50\x3d\xdf\xa3\x5e\xff\xa1\x48\xc6\x5d\x4b\xa1\xda\xf5\xe5\x02\x24\xbb\xc2\x7a\x31\x12\xba\x44\x0e\xb1\x93\x20\x0d\x5f\x08\x48\xc3\x87\x83\x34\xac\x03\xe9\x5c\x60\x0e\x94\xc9\x8c\x9e\x42\x96\x85\x45\xa0\xa9\x8d\xf2\x4d\x02\x37\x7a\x21\xc0\x8d\x1e\x0e\xdc\xa8\x0e\xb8\x63\x96\x5b\x8b\x37\x44\x2e\x12\x1a\xfa\x70\x1f\xf0\x2d\x11\x52\x94\x3b\x0c\xcf\x15\xba\x27\xb5\xff\x39\xe8\xea\x3c\xa3\x7a\x33\x0e\x45\x6e\x05\xca\x4d\x48\xac\x16\x6d\xec\x60\x89\x0b\x2d\xbb\x79\x55\x63\xdc\xff\x37\x22\xe5\x4c\xd9\x67\x65\xd8\x8d\x29\x4f\x2c\x9b\x19\xe3\x26\xe2\x13\x3b\x01\x88\x27\x00\xec\xff\xaa\x3b\x23\xdb\x25\x94\x08\xc9\x91\x54\xac\xcf\x1e\x6a\xf1\x01\x06\x66\x40\xd3\x57\x91\xb3\x0e\x88\xda\x86\x3a\x32\x03\x22\x75\x38\xc4\x11\x4c\x6d\xa6\xe4\x23\x3e\x55\xbc\x17\x23\x74\x0c\xd7\x3e\xe6\xab\xc4\x3c\x53\xe4\xe2\x31\x20\xb1\xa2\x56\xd9\xec\x7f\xc6\x7c\xc6\xb8\xab\xbf\x88\x2c\xe3\x2b\x51\x40\xd4\xf4\x5a\x70\x46\x99\x2f\xc0\x45\x94\xea\xd8\x47\x95\xd4\xcb\x95\x87\xc7\x30\x65\xcc\xc1\x88\x26\xde\xa8\xf9\x27\x1c\xdb\x63\x90\xdc\xc7\x95\x1e\xd2\xa0\xdc\x81\xdf\xd7\x82\x91\xb2\x18\x2f\x63\xf5\x8e\x7a\x3d\x4d\x3b\x61\xf4\xe1\x0a\x30\x3b\x44\x79\xdc\x44\x99\x55\x23\x47\x66\x87\x98\xdf\xe8\xbc\x3a\x24\xaf\x0e\xc9\xab\x43\xa2\x1c\x12\xa3\x53\x1e\xe1\x96\xa4\x06\xf8\xcb\x3a\x27\x8f\x83\x31\x3b\xc0\xc3\x1d\x95\xd0\x05\x31\xc3\x55\xbb\x20\xcd\xfc\x1a\x0f\x49\x6b\x31\xce\x8e\x7f\xee\xd9\x48\xe2\xc4\xf0\x61\x3e\x23\x15\x4f\x7d\xf5\x58\xa2\x4f\x35\x73\x21\x53\x7e\xa0\xaf\x31\xce\xfb\x81\xc1\x4c\xbe\x65\x76\x62\xb4\xb4\x98\x98\xd9\x09\xc8\x9c\x85\xb4\x44\x8d\x0b\x16\x52\xf5\x32\x2a\x5e\x44\x0d\x22\x2e\x86\x92\x5c\xdc\xe5\x1e\xfe\x50\x45\x40\x33\x94\x3d\x03\x54\x36\xe8\xf0\xe2\xc2\x4b\x9f\x99\xf8\xde\xf1\xa5\x9c\xc7\x98\xc2\xf3\x2d\xb2\x43\xe1\x7a\x06\x5a\x37\xe7\xa3\xdd\xc3\x75\xf9\xa9\x54\x0f\x2b\x42\xf0\xc2\xa4\x84\x13\xee\x84\xa8\x75\x27\x7e\x2a\x33\xa3\x72\x66\x12\x36\xdd\x04\xfd\xb4\x45\x4f\x6b\xc1\x9f\xb3\x1e\xb3\x6c\xbc\xc8\xb4\x48\x7e\x8f\xdb\x28\xdb\x58\x96\x52\x30\x83\x78\x4c\x14\xa7\x13\x2c\x8e\x63\x63\x53\x64\xbe\x0f\x90\xb5\x80\x60\x30\x9d\xee\x40\x20\x08\x9d\x3b\x85\xd6\x50\xd9\xd0\xcc\x7b\x65\x5c\xbb\xa0\x63\xcb\xd8\xd4\x87\xdc\x44\xe2\x23\x17\x48\x1b\x5a\xd5\x52\xab\x71\x25\x44\xaa\x83\x31\xc6\xa9\x91\x7d\xb9\xc0\x54\x2a\xdc\x23\x7f\x01\x87\x5a\xeb\xdf\x2c\x3c\x50\x67\xe6\x43\xf8\x12\xc5\x01\x3f\xd2\xb6\x87\x66\x0b\xad\x1c\x86\xec\xb4\xd5\x2b\xb3\x79\xe7\xa7\x27\x78\x4e\xf2\x6b\xa5\xc6\xac\x85\xdd\x4a\x12\x26\x07\xe7\x0f\x1a\x35\xec\x96\x1b\xf5\xc1\xe1\x9a\x57\x8f\xa3\x2e\xe2\x60\x59\xd8\x7b\xa9\x31\xac\x30\x31\xf6\x88\x18\x56\x66\x88\xd7\x18\xd6\x6b\x0c\x2b\x04\xe9\x89\x63\x58\xd1\xb0\x47\xe8\x76\xd7\x71\xd8\x0d\xb6\x0f\x83\x3d\xfa\x89\xa9\x51\x78\xc4\xf7\xea\xc6\x2c\x24\xe4\x0c\x73\x57\x1c\x33\x19\xea\x80\x47\x7c\xbf\x64\xa8\xea\x18\xde\x8c\xf1\x29\xb1\x6d\x4c\x01\x13\x5d\xcd\x31\xc5\x16\xf2\x05\x8e\xbd\x8d\xb4\x67\x5e\x1a\xe8\x03\x96\xee\x1b\x56\x85\x50\xdf\x9d\x9a\x0d\x77\x5c\xdd\xa9\x5d\x1b\x0b\x51\x98\xe2\xc0\xc7\x0a\x1c\x1c\x22\xcc\x37\xb3\x95\x23\xdd\x72\xef\xfb\xf9\xca\xee\xf7\xcc\x6b\x9e\xc5\xfe\x1d\xb6\xa3\xe2\x1c\xb0\x19\x16\xb4\x2d\x4d\xd0\x30\x89\xd9\x9b\x17\x82\xd9\x9b\x63\xe4\xe2\x3d\x46\x67\x0e\xb1\xe4\xc3\xf1\x2b\x1a\xa6\x5c\x59\x2a\x3c\x74\xcb\x58\xee\x6c\x2c\xcd\xee\x27\x28\x02\xb2\x02\x13\x65\x42\x5a\x44\x44\x90\x97\xef\xa7\x9e\x2b\xc8\xdf\x37\x6b\xbc\x4b\xc1\x2f\xdb\x3b\xc2\xcd\x82\x38\x21\x96\x74\xae\x81\xcd\x44\x62\x83\x61\xef\x99\x5b\x8e\x37\x50\x45\xc3\x25\x8a\xc5\x0a\xb2\xd1\x61\x85\x65\xa6\xa7\x28\xda\xee\x7d\xa2\xce\x0a\x78\x54\x1f\xc7\x04\x0e\x37\x7f\x81\x4e\x43\x1c\xa7\xf7\x6b\x45\xe1\x50\xb3\x87\x6b\xb2\x65\x2b\xa9\x6d\x13\xf7\x42\xe9\xde\x11\xc4\xdd\x3a\x4c\xe0\xe7\x39\xf5\xd9\xfa\x5a\xb8\x87\x63\xaf\xfa\x3e\x8d\x43\x9f\x18\xe9\x65\x86\x11\xeb\x80\x3b\x34\x1e\xe3\x17\x1f\xf3\xd5\x23\x1c\xfb\x23\xe4\xcc\x18\x77\xb1\xfd\x19\xcd\xf1\x9e\xcf\x05\xe3\x45\xb0\xbd\xc0\x88\x66\x1d\x80\x4f\xec\xde\xff\x75\x3d\xf6\xc7\x66\x9d\x5f\x64\x94\xb3\x09\xd2\x4f\x6a\xc6\xab\x8e\x71\xb4\xcb\x02\xab\x14\xb9\x89\xa9\xaa\x6d\xae\x23\x9e\x13\x65\xfc\xee\xd1\xc9\x43\x73\x3c\xb1\xb4\xe6\xb8\x47\x2f\x41\xee\x34\x65\x99\xa0\xaf\xc3\x7c\x7b\xe2\x71\xb6\x24\x36\x2e\x38\x6a\x52\x79\x00\x43\xf8\x9e\xc7\xb8\x92\x14\x3d\x0c\x44\xc3\x94\x99\x6a\xd5\xea\x73\xa6\xd1\x83\x0d\x76\x7b\xd0\xeb\xb5\x4b\xc5\xd8\xd0\x8b\xed\xc6\xc4\xc2\x0f\x35\xe0\x49\x24\xd2\xf6\xbb\x3d\xea\xf5\xcb\xd9\x7a\xd5\xfd\x06\xa4\xcd\xaa\xb9\x7f\x55\x61\x8f\x51\x61\xdf\x51\xbd\xe8\xd3\x2a\x1b\x5c\xc7\xd2\x1f\xac\x6b\x82\xee\x51\x21\x45\xc9\xba\x6e\xa2\x83\x4c\x54\xff\xb9\x68\xa2\x90\xb3\x9f\xa6\x90\x0c\x1c\xaf\xea\xe8\x55\x1d\x45\x3f\x3f\x4c\x1d\xd5\xa4\xaa\xd3\x8d\xbf\x93\xee\x6a\xd8\x3c\x8c\xd9\x4e\xe4\xca\xc3\xed\xb5\xb5\xb8\x95\x62\x2b\x00\xda\x70\xf8\x69\x7a\x89\x2d\x79\x82\x67\x98\x63\x6a\x45\x18\x9a\xf4\x2e\xd3\x2f\x43\x6c\xb8\xd2\x54\x92\x24\x27\x81\xd8\x49\xd0\x4d\x27\x21\x39\xa1\xf3\xe8\xf1\x15\xa1\xf5\x8d\x16\x8a\x9f\xea\x46\x51\x22\x33\x01\x43\x62\x25\x77\xf4\x87\x82\x5f\x6d\xa2\x84\xdb\x25\x14\x49\xc6\xe3\x0e\x01\x07\xab\x63\x9d\x29\x57\xed\xe3\x2f\xea\x46\x4a\xb1\x8c\x2b\xbe\x97\xf8\x84\xfa\x55\xcd\x4a\x92\x1a\x89\x5d\x71\x3f\xfc\x1a\x81\x43\xf1\xad\x9c\x24\xe4\xa6\xb6\x83\xa2\x2b\xdf\x88\x50\x89\xe7\x91\xd5\x69\x0c\x91\x7e\xa5\xd7\x64\xd8\x0a\x39\xce\xa7\x59\xdd\xfa\x08\x57\x73\x46\xbe\x92\x2b\xa5\x00\xa3\x32\x9c\x40\x2b\x20\x3b\xb7\xc4\x0b\xd9\x37\xb3\x87\x0a\xd4\x51\x69\xf3\xc8\x06\x4f\xd2\x12\x5d\xd8\x29\x3a\x32\xfe\x20\x40\x54\xc7\x47\xa0\xa0\x85\xac\x98\x44\xc4\x39\x5a\x65\xde\x14\x36\x87\x4a\x02\x35\x7b\x86\xc2\x74\xfc\xec\x07\xcd\x7f\x7e\xe1\x99\xe6\xae\xef\x48\x32\x41\x77\x99\xc7\x66\x83\x9a\x8c\xd8\x36\x40\x31\x79\xac\x3e\xfe\x49\xd9\xc4\xd6\x1f\xc8\xf1\xb1\x18\xc3\x3f\x50\x90\x99\x5b\x07\x8f\x63\x0f\x29\x39\x58\x37\x2e\x91\x20\x8c\xea\xdf\x38\x46\xf6\x6a\x1d\x66\xfa\xd0\xe9\x3a\xd8\x38\x7a\xbd\x6e\xce\x40\x11\x3a\xff\x27\xb4\x9a\x8a\x63\xda\x29\xad\x26\xf3\x58\x1f\x74\x9e\x81\xf6\x8f\xc0\x0f\x6a\xf0\x6c\xec\x39\x6c\xd5\x85\x77\x8c\x87\xb6\x13\x76\x2f\x4e\x1b\x53\x10\x82\x5d\x2c\x69\xf9\xb2\x1f\x08\x5c\xc3\x26\x90\x46\x17\xf0\x24\xfc\xe4\xa0\xcc\xce\xca\x38\x9c\x29\x06\xc6\xe0\x8b\x0e\x46\x42\x76\xfa\x3a\xf4\x7e\x1f\x7e\xe2\x30\x46\x73\x9d\x90\x13\xac\xc6\x5d\x75\x29\x56\xd3\xc6\x8b\x44\x1d\x5b\x6d\xe3\x20\xef\x3a\x41\xb9\x2e\x33\xc6\x5d\x24\xc7\x60\x23\x89\x3b\x92\xb8\xb8\xe9\x90\x41\xc9\xf1\x53\x0e\x69\xd6\xc1\xe4\x9e\x4a\x38\xbc\xbd\xa8\x69\xfb\x94\x97\xd3\xb0\x17\x2e\x72\x15\x8b\xf4\x57\x75\x51\x50\x91\x7a\x7c\xd9\x16\x21\x43\x71\xfc\x53\x4d\x7b\x1a\x02\xc3\x45\xe6\x52\xa0\x1f\x64\x36\x0a\x67\x56\xfb\x57\xd0\xda\xfd\x7c\x18\x10\x95\x56\x18\x44\xbd\x5c\xf6\xd3\x0f\x17\x86\xac\x54\x98\xa0\x95\xf1\x44\x1c\x07\xeb\xf2\xc9\x1c\x90\x1d\x33\x66\x98\x5f\xcc\xea\xa7\xe2\xd1\x37\xca\x9b\x07\x0c\x04\x30\xa7\x25\xa5\xca\x55\x2a\x25\xf0\x47\x09\x46\xe1\x04\xa6\x6e\x81\x09\xc7\x4c\x5f\xcf\xa5\xbb\x47\x99\xd5\xb0\x4a\x25\xb8\xd1\x24\x8c\x6c\xc0\x94\xd9\x21\xf9\xb9\x79\x4f\x57\x09\x9b\x1f\x17\xdd\x4e\xc2\x0b\x4e\x26\x41\x75\x4a\xaa\x22\xb2\xa1\xbf\x5e\x34\x76\xae\xc2\x43\x5f\xa2\x16\x57\x77\x20\x8f\x04\xf4\xe7\x5c\xf2\xe6\xbe\x4f\x01\xfd\x0d\xe4\xa0\x90\xed\x2a\x0b\x7d\x48\x6d\x9d\xdb\x7e\xf8\x35\x31\x69\x06\x93\x8e\x42\x61\xa9\x6d\x13\x21\x88\x8e\x7b\x31\x7b\x05\x02\x53\xa9\x1c\x9c\x68\xcd\xc0\xe7\x4f\xa7\x67\x15\x7b\x36\x65\x8b\xef\x39\xc9\xa5\xde\x57\x6e\xaa\x33\xe1\xb1\x1b\x7d\x67\xa0\x8c\x64\x20\xcc\xc4\x47\xeb\x23\x72\x7d\xc2\xfa\x29\x42\xeb\x36\x73\x45\x9e\x58\x1a\x2d\x2c\x4d\x71\x8b\x64\x5a\xa2\xd5\xbf\x16\xa3\x33\x32\xf7\x6b\x88\x91\x4c\x91\xa2\x3f\xb0\xfb\x67\x8e\x8e\xac\x93\x97\x75\x6a\x52\x44\xb4\x15\x1a\x34\x70\x45\x2b\xbe\xd9\x85\x43\x09\xae\x2f\xa4\x22\x51\x04\xe1\x46\x87\xdd\x60\xde\xb1\x90\xc0\x80\x1c\x6f\x81\xa8\xef\x62\xae\x7c\xc1\x05\xe2\xc8\x92\x98\x0b\x60\x1c\xda\xed\x4e\xbb\xbd\xae\x96\x12\x0f\x4e\x5c\x22\x6a\xda\x4f\xb1\x4c\xb6\x36\xe7\xde\x70\x78\x8a\x23\x6c\x95\x1b\xd5\xb4\xb3\x10\xd5\xc9\xc9\x29\x06\x87\xd1\x39\x56\x32\x87\x28\x0c\x07\x89\xcf\x77\xdb\x75\xb3\x94\xf7\x7e\x0b\x0a\xbf\x54\x93\x27\x97\x91\x5c\xa2\xe6\x7b\xfa\x20\x55\x46\x36\x47\x48\xda\x9a\xe9\xa0\x04\xb4\x32\xa6\x54\x19\x9d\xac\x46\xaa\x2e\xe9\x48\x7e\xa5\x20\xf3\xfc\x3c\xbd\xa4\x24\xd1\xad\x78\xda\xe2\x70\xf6\x4f\x9d\xb4\x98\x8c\xef\x37\x65\xa5\x87\x1e\x9e\xef\x84\x19\x92\x5b\xf9\x55\x56\x68\xb4\xda\x7b\xe9\x9d\x6b\xbb\xc2\xc2\x64\xc3\x7e\xe9\x81\x62\x23\xac\xb4\x84\x62\x39\xaa\x5f\x34\x73\xdb\x85\x8b\x40\x45\xb4\xdb\x29\xc2\xda\x6d\x70\x08\xbd\xaa\x57\x57\xa4\xe2\xf3\xe7\x94\x5c\xfb\x18\x88\x8d\xa9\x24\x33\x82\xe3\x53\xc9\xe6\xe3\xb5\x83\xdb\x44\x78\x0e\x5a\x4d\xaa\x0d\xc6\x71\xc2\x58\x64\xcc\xa8\x32\xfc\xc1\x20\xe0\xf9\xdc\x63\x02\x37\x50\xc1\xd5\x9f\xfb\xe0\xbb\x88\xc2\x8c\x13\x4c\x6d\x67\x55\xc0\x5d\x9a\x86\x75\x4d\x44\x18\x39\xf9\x17\xba\x11\xff\xaa\xa7\x00\x53\x34\x75\x70\x05\xb4\x17\x81\x53\x55\xc0\x33\x11\x61\x77\xc3\xbe\x8e\xdf\x10\x3a\x57\xe6\xeb\xd3\xe9\x3e\x58\xe6\xae\xde\x02\x22\xd2\x76\xba\xc8\x05\x0a\x06\xce\xea\x9d\x62\x31\xde\x8f\x7f\x53\xd0\xa0\xd0\x6e\xe9\xff\x5b\x3f\x4f\xc6\x0d\xcd\xed\xf6\x8b\x13\xee\x00\xbf\x22\xa1\xce\x48\xd9\x71\x17\xfe\x20\x7c\x4e\x28\x41\x4f\x2d\x6d\x01\x11\x4f\x25\x65\xe6\x63\x33\xe4\x3b\x72\x0c\x33\xe4\x88\x38\x34\x14\xe5\x8e\x27\xa9\xf8\x4c\xf9\x96\xa9\x7d\x56\xe1\x0b\x45\x77\x31\xe8\x31\x12\x89\xe9\xf0\xe0\x81\x61\xac\x80\xe0\xac\x91\x30\x06\x02\x52\xa6\xa0\x61\x9a\x29\xb9\x7e\xcc\x93\x32\x26\x93\x1b\x9d\xd4\x7d\x01\xf7\xcc\xab\xfd\xa2\xa7\x87\xb2\x9b\xc4\x89\x53\x35\x79\x32\xbc\x6e\x22\x06\x62\x46\xb0\x63\x66\xd3\x04\xf2\xa2\x31\x8a\x43\x9e\x25\x71\x31\xea\x3b\x8e\x62\xd2\x6c\x8a\x61\x6d\x2d\x9f\x0b\x8d\x97\x95\x39\x69\x5a\x7c\x35\x86\x9a\xcc\xc3\x7d\x73\xf7\xa8\xc5\x78\x74\x9b\x47\x26\x11\x5c\x40\x06\xa1\x63\xf0\x90\x5c\x64\x27\x22\xde\xa6\xcf\x88\x93\x4c\xcd\x1a\x32\x82\x87\x89\x41\x92\x07\x63\x0b\x6f\xf2\x75\xb1\xe4\xc4\xd2\x1b\xa6\x19\x96\xd6\xa2\x0b\xef\xd4\x3f\xfa\x02\xf8\xf0\xdd\xcd\x02\x53\xc0\xae\x27\x57\x5d\xd3\x4f\x4d\x2a\xc1\x42\x17\x8b\xc7\x37\x85\x48\xcc\x29\x0a\x7b\x69\x8a\x44\xb7\x92\xe5\xb4\x58\xe6\xbc\x96\x62\x9d\x13\xae\xb4\xf8\x22\x93\x5c\x32\xd0\x80\x91\x78\xd1\x04\x90\xcf\x68\x8e\xc1\x34\x5f\x0f\xb5\x7a\xb0\x6b\x52\x9b\x7a\x3d\x1c\x10\x0a\x7a\x6b\xe6\x32\x8e\xf5\x13\x83\x01\xc7\x2e\x22\x94\xd0\xb9\xbe\x2d\xdf\x25\x42\x04\x87\x03\xcc\xd1\x0b\x73\x89\xbf\x19\x40\x04\xd5\xf7\xf1\x89\xa0\x78\x72\x93\xca\xa3\x81\x90\xe4\xeb\x00\x4a\x12\xa3\x4b\xe4\xf8\xca\x79\xfd\xb4\xff\xf6\xe0\xb8\x77\xb2\xff\xe5\x6e\x2e\x8f\xef\x0e\x36\x8f\x4e\x7b\xbd\x2f\x77\x47\x83\x8f\xe7\xbf\xef\x7d\xb9\x9c\xcb\xe3\xb3\x77\x07\x5f\xee\x0e\xef\xbe\x5c\x2e\xf6\x3e\x9d\xbf\x3b\xd8\x9b\xff\xf6\x9b\x71\xf9\x92\x59\x54\x83\x6e\x22\xdf\x5b\x09\xeb\x71\x7c\x78\x4a\xcd\xaf\x42\x30\x82\xf3\xe9\x01\xc8\x66\x7b\x43\xce\xfb\xbd\x9e\x61\x24\x69\xaa\x0c\x23\x89\x98\x48\x35\x23\x29\x33\x96\x2c\x83\x7f\x20\xfd\x79\x9d\x64\x08\x8a\x9f\x37\x21\x4b\x17\x14\xeb\x93\x1b\xec\x26\x28\xc2\x7a\x12\xea\x0a\x13\x0a\x81\xb2\x4b\xbe\x2a\xd3\x7b\xb9\x0b\x84\xb4\x5f\xa3\xb4\x8c\xd6\x51\x01\xa1\xc2\x77\xa4\x80\xe9\xaa\x84\xd3\x06\x32\x50\xcc\x5e\x5e\x32\xf0\x12\x39\x93\xd2\x2c\x49\x28\x27\xaa\x55\xbc\xc3\x53\x8d\x6d\xc4\xed\xfa\x7e\x61\x4b\xd5\x37\x2c\x4c\xd3\x91\xbb\x90\x84\xa0\x32\x2d\xc9\x97\x72\x24\xf4\xd3\xe0\xa1\xf9\xe5\x5d\x90\x67\xfa\xfd\xe2\x2c\x65\x26\x17\x52\x7a\x6b\x59\xc6\x32\xe7\xe7\xc3\xe1\x0d\x55\x11\x99\x41\x3c\x06\x5a\x51\xca\x30\x66\x31\x13\xdf\x83\x16\xba\x49\x44\xde\xcd\x7c\xb7\x82\x78\x27\xf2\x88\x8c\x12\x1b\x99\x43\xf6\x75\x9f\xc6\x7e\xe7\x06\x3f\xd1\xa7\x0b\xb3\x50\x25\x04\xe8\x0c\x44\x9f\x9c\x7e\xdb\x3a\xf9\x32\xfc\xfd\xef\x87\x3b\x5f\x7a\x9f\xce\xdc\xcb\x2f\xef\xec\x21\xb3\xde\x9d\xcc\x5b\x69\x1f\x3d\xca\x56\xc4\xcf\x2b\x93\x14\x1b\x8d\x86\x0e\x32\xec\xd0\xd2\xa9\xf1\xa6\x08\x44\xc1\xce\x64\x6c\xbe\x7a\x36\x13\x2a\x05\x5a\xc8\x21\xc9\x64\x51\x41\x0a\x17\x5a\xee\x4a\x3d\xa9\x40\x3d\x81\x03\x13\x32\xf5\xaa\xd3\x27\x62\xb5\xc5\xaf\x87\x97\x57\x64\xe7\xba\xc7\xa4\x7b\x79\x3d\x53\xbc\xcf\xf8\xbc\x1b\xc8\x69\x57\x23\xd6\xb5\x98\x9b\xe0\x3a\x4e\xdb\xea\x9b\x16\x7a\x9d\x7e\xaf\xd3\xdb\x3c\xeb\x0f\xc6\x9b\xfd\xf1\x60\xd4\xed\x6d\x0e\xfb\xa3\xc1\x9f\x71\x8f\x44\x56\x36\xd7\x63\x6b\x3c\xdc\xea\x0e\xb7\x06\x83\xde\x4e\xa2\x47\xf4\xc7\x5f\x5a\x83\xee\x56\xb7\x17\xbf\x48\xaf\xe4\x68\x85\x17\xc8\x55\x83\xeb\xb8\x9f\xb5\xac\x99\xbc\xf3\x8f\x12\xb6\x29\x9b\xfe\x45\x45\x2d\x9d\xdf\x87\x16\x0a\xea\xad\x8a\xce\x86\xc6\x87\x70\xb3\xb3\xd3\x5c\x2e\x0b\xce\x00\x66\xe4\x30\x9d\x27\x4d\x74\x6b\xa5\xfd\xa4\x54\x34\x33\x57\x58\xf7\x18\x77\x11\x8a\xdc\xf9\xce\x3d\x16\x48\xd5\x22\x79\xa2\x85\x02\x15\x8a\xb9\x7e\xbd\x54\xac\x99\xba\x75\x03\x35\x8a\x1a\x9a\xae\x20\xa8\x5e\x45\xf0\x94\x2b\x09\x1e\xb4\x9a\xe0\x41\x2b\x0a\xaa\x14\x38\xd4\x2d\x96\x82\x9c\x48\xcd\x3a\xc9\x27\x24\x20\xd4\xe6\x45\x2e\x49\xea\x59\x2a\x04\x06\xad\x5d\x17\xdd\x31\x0a\x17\x78\x1a\xe6\xf7\x13\x6d\xc3\x20\x55\x2c\x2e\xf9\x6c\x40\x03\x52\x93\xa1\xf8\x88\xd0\x02\x39\xcb\x90\x76\x7e\x0a\x07\x48\xc8\x75\x48\x44\xd6\xaa\x68\x83\xaa\xf8\x15\xfc\x23\x76\x79\xd7\x03\xb7\xd9\x6c\xc7\x8f\x4c\xcc\xe0\x04\xd1\x39\x2e\x3a\x7c\x9b\x61\xcc\x04\x0b\x92\x6a\x62\xa2\x09\x9e\x4c\xc6\x61\x81\xc1\xc4\xfc\x79\xb2\xc9\x94\xb3\x2b\xcc\x25\xf3\x88\x15\x04\x26\x26\xd3\x95\xc4\x62\x42\xe8\x44\x32\x99\xb8\x07\x33\xaa\x4d\x98\x98\x5a\x56\xc6\x27\x84\x4d\x82\x78\x5e\x34\x6e\x27\x7f\x21\x9f\x1e\x7c\x0c\x93\x89\xc5\xa8\xf0\x5d\xb5\x04\x67\x33\x81\x13\xc7\xc6\x97\xa6\x10\x32\xa5\xd6\x24\x71\xb1\x90\xc8\xf5\xc6\xd0\xdf\xea\xf7\xb7\xb6\x7b\x83\x61\x2f\xfc\x8b\x69\xa9\x8e\x63\xd8\x19\xf5\x37\x47\x75\xbd\xb7\x4a\x7b\x6f\xee\xec\xec\xd4\xf5\x7e\x53\xda\x7b\x7b\x6b\x30\x48\x4e\x92\xb9\xe4\x45\xfe\x1b\x4e\x53\xed\x94\xe4\xa6\xa3\xfc\x2e\xf3\x42\x2f\xaf\x37\xcc\x79\x73\x89\x5a\x61\x28\x33\x51\xc1\x5f\x6b\x4b\x75\xd7\xb5\xdc\xd0\xda\x3b\x38\x3e\x3b\xd9\xfd\x78\xda\x39\x7a\x7f\x74\xd6\x49\xb5\x88\x7c\x8b\xd3\xc4\xcd\x74\xe1\x9d\x75\xe6\x6e\x9b\x68\xa5\xae\x83\x2f\xb0\xb9\xc3\xee\x37\x1d\xbe\x8c\x42\x97\x09\xff\x2c\x59\xe1\xad\x2c\xf2\xc5\x21\x71\xaf\xdf\x5b\x7c\xdf\xff\xb8\xd5\x47\xe7\xb7\x87\x7f\x5e\xbf\x3d\xbb\x3e\x3e\x09\x14\x44\xf9\x1d\x59\xaf\xd8\x54\xde\x11\x50\x05\xd3\xe0\x51\x28\x0d\xea\x40\x1a\x14\x61\x14\x51\x6a\x82\x91\xc6\xd3\x2b\xe7\xdd\xb9\xd8\x7f\xef\xaf\xa6\x87\xfc\x80\xde\xf2\x5d\xec\x6e\x0f\x46\xf3\xeb\xab\x2b\xb2\xbf\x8c\x79\x37\xb1\xce\xcf\x21\x90\x4d\x18\xef\x3f\x8a\xf1\x7e\x1d\xe3\xfd\x02\xc6\xc3\x90\xac\x87\xe4\x22\x9e\xf5\x48\xc7\x40\xf2\xd8\xd2\xfd\x51\xc8\x5e\x93\x54\xc4\xf5\xf6\x63\x98\xde\xae\xe1\x79\xbb\x80\xe5\x26\xf7\x2b\x45\x0c\x14\xde\x73\xfe\x2c\xd8\x88\x76\x31\x01\xfd\x3a\x20\x4f\xec\xdf\xda\x7d\xf2\xf7\xa1\xed\xff\xf1\xed\x70\xb9\xdc\xfc\xb6\xfc\xe8\xac\xee\xfa\xee\xfb\x93\xe1\xef\xab\xeb\xe3\xb6\x5e\xf6\xfa\xf2\xda\x8a\x85\xfd\xed\xd3\xf6\x7c\x30\xdf\xfa\x70\x66\x9f\xff\xfd\x1c\x0d\xae\xc4\x87\x9d\xc1\xd5\x97\xfd\xe1\x2a\x44\x25\x7b\xe8\xb0\x50\xd9\x3d\x4a\x98\xfb\x75\xc2\xdc\x2f\x12\xe6\x73\xed\xab\x81\x64\xca\x4b\x26\xb3\x15\xfc\x7e\x71\x66\x8e\x75\x8e\xe1\x24\x08\x87\x46\xf7\x4b\x98\xed\xa6\x39\xf4\xd9\x08\x95\xe1\xf9\xe2\x60\x71\xe3\x7e\x7d\xeb\x5d\x7c\x9e\x1d\x0e\x9c\x63\x7c\xe5\xd9\xa3\x3f\xf7\x43\x54\xb2\xb7\x63\x14\xa1\x32\x7a\x0c\x28\xa3\x1a\x4c\x46\x45\x90\x08\xcc\xa1\x3d\x65\xd3\x76\xd3\xcb\xde\xba\x15\x0b\xfe\xdb\xf0\x9c\x1c\x2c\xee\x68\x02\x83\x4b\xcf\x1e\x7d\xdb\x8b\x30\x68\x7a\xf9\x5e\x11\x38\x9b\x8f\x01\x67\xb3\x06\x9c\xcd\x6a\x70\x16\x48\x84\x35\xaa\x80\x0a\x2e\xb8\xdb\x32\x7f\x62\x5a\x17\xda\x05\x57\xdd\xd5\x02\x75\x75\xab\x80\xfa\xe3\x33\x3e\x1c\xb0\x63\x7c\x69\x0f\xbf\xbe\x8d\x70\xaa\xb9\x1b\xb0\x70\x45\x0d\x1e\xb5\xa2\x06\x75\x2b\x6a\x50\x80\x50\xb4\x6a\xa4\xa2\x17\x16\x68\x89\x83\x3a\x48\x4c\x21\x3c\xf2\x54\x8a\xc3\xd5\xd7\xbd\xbb\x0b\xcd\x7e\x88\xc3\xc7\xe5\xbb\x37\x97\x47\x5f\xbe\x85\x38\x54\xdd\x09\x57\x84\xc1\x70\xeb\x31\x18\x24\x7b\x17\x62\x90\x6c\x90\x55\xb4\x89\x7a\x03\x5d\xc2\xaa\xff\x4e\x82\x0e\x8f\xe8\xf3\x54\xa5\x20\x6c\x5d\x7d\xeb\x29\x61\xb8\x8b\xd1\xf8\x86\x17\xf6\xf0\x20\x50\x1c\xf9\xa3\xc9\x45\x8c\xbf\x79\x0c\xdf\x6f\x6a\xd8\x7e\x53\xa8\x4b\xe3\x4b\xcd\x71\xfa\x53\x39\xd5\x88\x0f\xc2\x79\xdd\xfa\x36\x5f\xcc\x8e\xde\xcc\xdf\x9f\x88\x0f\xcb\x83\x8b\x88\xc3\xc6\x66\xf4\x87\xf3\x69\x8e\xb2\x87\xc7\xef\x40\xb9\xc0\x02\xcb\x31\x7c\xda\x3b\xea\x1c\x7c\xed\xbc\x19\x07\x11\x49\x73\x5e\x4e\x71\x11\xb7\xc1\xb7\xb2\x93\x8a\x20\xdd\xf6\x86\x0e\xb5\x1d\xf7\xba\x77\x3d\xb3\xb6\x05\x91\x68\x53\x38\x97\xcb\x9d\xe4\x1e\x6d\xc6\x78\x58\x41\xa3\x59\xee\xcf\x37\xed\x9d\x9d\xeb\x9e\xc3\x2d\x7b\x39\x9a\x6f\x23\x67\xba\x2d\x9c\xd9\x9c\x5e\x0e\xed\xc5\x54\x5c\xfe\xd7\x7f\xfc\xed\xe0\xeb\xd9\xc9\x2e\xfc\xb7\xe1\xb2\xab\x01\xf9\x2d\x2e\x4e\x4a\x8c\x4d\x04\xb4\x47\xbd\x51\x7b\x5d\xf3\xaf\x7f\xdd\xfb\x78\x7e\x7a\x76\x70\x12\x9a\x88\xde\xa8\x6d\xee\x18\x8f\xff\x4c\x6a\x5c\xe5\xa4\xda\xf7\xe7\x9b\x8c\x6f\xf6\x96\xc4\xef\x6d\x33\xac\x66\x68\xc1\xaf\xac\xc1\x96\x3d\x9f\xc9\xcb\x3e\xb2\x52\xf7\x03\x84\x7f\x1b\xbe\x5d\xc7\x44\xc2\xf1\xf8\xb5\xca\xc6\x9e\x89\x0b\xbe\xda\xa2\xe2\x7a\x3a\x10\xc7\xee\xbb\xcb\xcd\xe9\x57\x6f\x7f\x7b\x0f\xb5\xd6\xfe\x3f\x00\x00\xff\xff\x62\x1f\x52\x73\x0e\x80\x00\x00")

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

	info := bindataFileInfo{name: "fleet-manager.yaml", size: 32782, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
