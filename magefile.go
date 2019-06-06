// +build mage

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
)

const (
	BINARY = "discord-set-slowmode-bot"
	DIST   = "release"
)

// create binary for current platform
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

	version := version()
	hubargs := []string{"release", "create", version, "-m", version}

	for _, t := range targets {
		fmt.Printf("Building %s-%s\n", t.goos, t.goarch)

		rOS := strings.NewReplacer("darwin", "macOS")
		rARCH := strings.NewReplacer("386", "32bit", "amd64", "64bit")

		archiveName := fmt.Sprintf("%s-%s-%s-%s.zip", BINARY, version, rOS.Replace(t.goos), rARCH.Replace(t.goarch))

		hubargs = append(hubargs, "-a", filepath.Join(DIST, archiveName))

		binary, err := build(t, DIST, true)
		if err != nil {
			return err
		}

		files := []string{
			binary,
			"README.md",
			"LICENSE.txt",
		}

		zipFiles(filepath.Join(DIST, archiveName), files)

		os.Remove(binary)
	}

	fmt.Printf("Creating GitHub release for version %s\n", version)
	err := exec.Command("hub", hubargs...).Run()
	if err != nil {
		return err
	}

	return nil
}

func Test() error {
	return exec.Command("go", "test", "./...").Run()
}

// cleanup binaries and dist directory
func Clean() error {
	log.Println("Cleaning...")
	err := os.RemoveAll(BINARY)
	if err != nil {
		return err
	}
	err = os.RemoveAll(DIST)
	if err != nil {
		return err
	}
	return nil
}

// helper functions

type target struct {
	goos   string
	goarch string
}

func build(t target, dir string, crush bool) (string, error) {
	os.Setenv("GOOS", t.goos)
	os.Setenv("GOARCH", t.goarch)

	binary := filepath.Join(dir, BINARY)
	if t.goos == "windows" {
		binary += ".exe"
	}

	err := exec.Command("go", "build", "-o", binary, "-ldflags", ldflags(crush), goflags()).Run()
	if err != nil {
		log.Fatalln(err)
	}

	if crush {
		err := exec.Command("upx", "-9", binary).Run()
		if err != nil {
			log.Println(err)
		}
	}

	return binary, nil
}

func goflags() string {
	if _, err := os.Stat("vendor"); !os.IsNotExist(err) {
		return "-mod=vendor"
	}
	return ""
}

func ldflags(crush bool) string {
	version := fmt.Sprintf("-X main.VERSION=%s", version())
	scrush := ""
	if crush {
		scrush += "-s -w"
	}
	return fmt.Sprintln(version, scrush)
}

func version() string {
	out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		log.Print(err)
		return "" // fallback
	}
	return strings.TrimRight(string(out), "\r\n")
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
