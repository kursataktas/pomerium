package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const envoyVersion = "1.30.1"

func main() {
	dir := outputDir()
	for _, target := range []string{
		"darwin-amd64",
		"darwin-arm64",
		"linux-amd64",
		"linux-arm64",
	} {
		download(target, dir)
	}
}

func outputDir() string {
	// This file is meant to be invoked via `go run`, in which case the stack
	// trace filename should give us the path to this source file.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("couldn't find current source file path")
	}
	return filepath.Join(filepath.Dir(file), "../files")
}

func download(target, outputDir string) {
	url := fmt.Sprintf("https://github.com/pomerium/envoy-binaries/releases/download/v%s/envoy-%s",
		envoyVersion, target)
	outputPath := filepath.Join(outputDir, "envoy-"+target)

	downloadIfNewer(url, outputPath)
	downloadIfNewer(url+".sha256", outputPath+".sha256")
	writeVersion(outputPath + ".version")
}

func downloadIfNewer(url, outputPath string) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "*/*")
	if fi, err := os.Stat(outputPath); err == nil {
		const format = "Mon, 02 Jan 2006 15:04:05 GMT"
		req.Header.Set("If-Modified-Since", fi.ModTime().UTC().Format(format))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("couldn't get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return
	} else if resp.StatusCode != http.StatusOK {
		log.Fatal("%s: %s", url, resp.Status)
	}

	log.Printf("downloading %s...\n", url)

	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		log.Fatalf("couldn't download %s: %v", url, err)
	}
}

func writeVersion(outputPath string) {
	output := make([]byte, 0, len(envoyVersion)+1)
	output = append(output, []byte(envoyVersion)...)
	output = append(output, byte('\n'))
	if err := os.WriteFile(outputPath, output, 0o644); err != nil {
		log.Fatal("couldn't write version: ", err)
	}
}
