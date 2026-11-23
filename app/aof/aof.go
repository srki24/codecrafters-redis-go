package aof

import (
	"bytes"
	"errors"
	"fmt"
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

var rootDir, appendOnly, appendFilename, appendDirName, appendFsync string

func initilizeVars(cfg *config.Config) {
	rootDir = cfg.Dir
	appendOnly = cfg.AppendOnly
	appendFilename = cfg.AppendFileName
	appendDirName = cfg.AppendDirName
	appendFsync = cfg.AppendFSync
}

func InitializeAOF(config *config.Config) error {
	initilizeVars(config)

	if appendOnly == "no" {
		return nil
	}
	dir := getDir()

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	aofFilename := strings.Join([]string{
		appendFilename,
		strconv.Itoa(version),
		"incr",
		"aof",
	}, ".")

	if _, err := os.Create(path.Join(dir, aofFilename)); err != nil {
		return err
	}
	manifestFile := manifestFile()
	manifestData := fmt.Appendf(nil, "file %s seq %d type %s", aofFilename, 1, "i")
	if err := os.WriteFile(
		path.Join(dir, manifestFile),
		manifestData,
		filePerm,
	); err != nil {
		return err
	}
	return nil
}

func getDir() string {
	return path.Join(rootDir, appendDirName)
}

func manifestFile() string {
	return appendFilename + "." + "manifest"
}

func getAOFfromManifest() (string, error) {
	dir := getDir()
	manifest := manifestFile()

	var aofFile string
	foundFile := false

	data, err := os.ReadFile(path.Join(dir, manifest))
	if err != nil {
		return "", err
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		fields := strings.Fields(string(line))
		if fields[5] == "i" {
			if foundFile {
				return "", errors.New("multiple active increment files found")
			}
			foundFile = true
			aofFile = fields[1]
		}
	}
	return aofFile, nil

}
func WriteCommanand(data []byte) error {
	if strings.ToLower(appendOnly) != "yes" {
		return nil
	}
	dir := getDir()
	aofFile, err := getAOFfromManifest()
	if err != nil {
		fmt.Println(err)
		return err
	}
	return os.WriteFile(
		path.Join(dir, aofFile),
		data,
		os.ModeAppend,
	)
}
