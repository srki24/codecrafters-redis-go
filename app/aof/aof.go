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

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
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

func InitializeAOF(config *config.Config, database *db.Database) error {
	initilizeVars(config)

	if appendOnly == "no" {
		return nil
	}

	info, err := os.Stat(getDir())
	if os.IsNotExist(err) || !info.IsDir() { // not exists initializ
		err := createManifestFile()
		if err != nil {
			return fmt.Errorf("failed to create AOF files: %v", err)
		}
	} else {
		err := loadAOF(database)
		if err != nil {
			return fmt.Errorf("failed to load AOF data: %v", err)
		}
	}
	return nil
}

func loadAOF(database *db.Database) error {
	aofFile, err := getAOFFPfromManifest()
	if err != nil {
		return err
	}

	reader, err := os.Open(aofFile)
	if err != nil {
		return err
	}
	for respRequest := range resp.ParseRequest(reader) {
		command, err := cmd.ParseCommand(respRequest)
		if err != nil {
			return fmt.Errorf("failed to parse command: %v", err)
		}

		response := cmd.GenerateResponse(command, database, nil, nil)
		if respErr, ok := response.(resp.SimpleError); ok {
			return fmt.Errorf("failed to apply command %s, error :%s", command.Name, respErr.Data)
		}
	}

	return nil
}

func createManifestFile() error {
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
	manifestFP := manifestFP()
	manifestData := fmt.Appendf(nil, "file %s seq %d type %s", aofFilename, 1, "i")
	if err := os.WriteFile(
		manifestFP,
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

func manifestFP() string {
	manifest := appendFilename + "." + "manifest"
	return path.Join(getDir(), manifest)
}

func getAOFFPfromManifest() (string, error) {
	dir := getDir()
	manifest := manifestFP()

	var aofFile string
	foundFile := false

	data, err := os.ReadFile(manifest)
	if err != nil {
		return "", err
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		fields := strings.Fields(string(line))
		if len(fields) < 5 {
			fmt.Printf("Malfored line, might be empty: %s", string(line))
			continue
		}
		if fields[5] == "i" {
			if foundFile {
				return "", errors.New("multiple active increment files found")
			}
			foundFile = true
			aofFile = fields[1]
		}
	}
	return path.Join(dir, aofFile), nil
}

func WriteCommanand(data []byte) error {
	if strings.ToLower(appendOnly) != "yes" {
		return nil
	}
	aofFile, err := getAOFFPfromManifest()
	if err != nil {
		fmt.Println(err)
		return err
	}
	f, err := os.OpenFile(
		aofFile,
		os.O_APPEND|os.O_WRONLY,
		filePerm,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}
