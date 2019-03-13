// +build mage

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/sirupsen/logrus"
)

const (
	BINARY = "discord-set-slowmode-bot"
	DIST   = "release"
)

var (
	Default = Run

	VERSION string
	LDFLAGS string
	GOFLAGS string

	err  error
	logr = logrus.New()
)

type target struct {
	goos   string
	goarch string
}

func init() {
	VERSION, _ = sh.Output("git", "describe", "--tags", "--always", "--dirty")
	LDFLAGS = fmt.Sprintf("-X main.version=%s", VERSION)

	if _, err := os.Stat("vendor"); !os.IsNotExist(err) {
		GOFLAGS = "-mod=vendor"
	}
}

func Run() error {
	return sh.Run("go", "run", "-ldflags", LDFLAGS, GOFLAGS, ".")
}

func Build() error {
	mg.Deps(Clean)

	_, err := build(target{runtime.GOOS, runtime.GOARCH}, ".", false)
	return err
}

func Release() error {
	mg.Deps(Clean)

	targets := []target{
		{"darwin", "amd64"},
		{"linux", "amd64"},
		{"linux", "386"},
		{"windows", "amd64"},
		{"windows", "386"},
	}

	for _, t := range targets {
		logr.Infof("Building for OS %s and architecture %s\n", t.goos, t.goarch)
		binary, err := build(t, DIST, true)
		if err != nil {
			return err
		}
		files := []string{
			binary,
			"README.md",
			"cfg.yaml.template",
			"LICENSE.txt",
		}

		rOS := strings.NewReplacer("darwin", "macOS")
		rARCH := strings.NewReplacer("386", "32bit", "amd64", "64bit")

		archiveName := fmt.Sprintf("%s-%s-%s-%s.zip", BINARY, VERSION, rOS.Replace(t.goos), rARCH.Replace(t.goarch))
		zipFiles(filepath.Join(DIST, archiveName), files)
		os.Remove(binary)
	}
	return nil
}

func Test() error {
	return sh.Run("go", "test", "./...")
}

func InstallDeps() error {
	logr.Info("Installing Deps...")
	return sh.Run("go", "mod", "download")
}

func VendorDeps() error {
	logr.Info("Vendoring Deps...")
	return sh.Run("go", "mod", "vendor")
}

func Clean() error {
	os.RemoveAll(BINARY)
	os.RemoveAll(DIST)
	return sh.Run("go", "mod", "tidy")
}

func build(t target, dir string, crush bool) (string, error) {
	envmap := envmap(os.Environ())
	if t.goos != "" && t.goarch != "" {
		envmap["GOOS"] = t.goos
		envmap["GOARCH"] = t.goarch
	}

	binary := filepath.Join(dir, BINARY)
	if t.goos == "windows" {
		binary += ".exe"
	}

	if crush {
		LDFLAGS = LDFLAGS + "-s -w"
	}

	err = sh.RunWith(envmap, "go", "build", "-o", binary, "-ldflags", "'"+LDFLAGS+"'", GOFLAGS)
	if err != nil {
		return "", err
	}

	if crush {
		sh.Run("upx", "-9", binary)
	}

	return binary, nil
}

func envmap(env []string) map[string]string {
	m := make(map[string]string)
	for _, pair := range env {
		splits := strings.Split(pair, "=")
		key := splits[0]
		val := strings.Join(splits[1:], "=")
		m[key] = val
	}
	return m
}

func goFileList() ([]string, error) {
	fileList := make([]string, 0)
	err := filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		if !strings.HasPrefix(path, "vendor") && strings.HasSuffix(path, ".go") {
			fileList = append(fileList, path)
		}

		return err
	})
	return fileList, err
}

// zipFiles compresses one or many files into a single zip archive file.
// The original code was published under MIT licence under https://golangcode.com/create-zip-files-in-go/
func zipFiles(filename string, files []string) error {

	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		zipfile, err := os.Open(file)
		if err != nil {
			return err
		}
		defer zipfile.Close()

		// Get the file information
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Change to deflate to gain better compression
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, zipfile)
		if err != nil {
			return err
		}
	}
	return nil
}
