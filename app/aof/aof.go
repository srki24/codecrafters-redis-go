package aof

import (
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/config"
)

const (
	dirPerm  fs.FileMode = 0755
	filePerm fs.FileMode = 0644
)

var version int = 1

func InitializeAOF(config *config.Config) error {
	if config.AppendOnly == "no" {
		return nil
	}
	dir := path.Join(config.Dir, config.AppendDirName)

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	filename := strings.Join([]string{
		config.AppendFileName,
		strconv.Itoa(version),
		"incr",
		"aof",
	}, ".")
	if _, err := os.OpenFile(
		path.Join(dir, filename),
		os.O_CREATE,
		filePerm); err != nil {
		return err
	}
	return nil
}
