package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sync/errgroup"
)

const envoyVersion = "1.30.1"

func main() {
	dir := outputDir()
	log.Println(dir)

	var eg errgroup.Group
	for _, target := range []string{
		"darwin-amd64",
		"darwin-arm64",
		"linux-amd64",
		"linux-arm64",
	} {
		download(&eg, target, dir)
	}
	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}
	log.Println("done")
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

func download(eg *errgroup.Group, target, outputDir string) {
	url := fmt.Sprintf("https://github.com/pomerium/envoy-binaries/releases/download/v%s/envoy-%s",
		envoyVersion, target)
	outputPath := filepath.Join(outputDir, "envoy-"+target)

	eg.Go(func() error { return downloadIfNewer(url, outputPath) })
	eg.Go(func() error { return downloadIfNewer(url+".sha256", outputPath+".sha256") })
	eg.Go(func() error { return writeVersion(outputPath + ".version") })
}

func downloadIfNewer(url, outputPath string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "*/*")
	if fi, err := os.Stat(outputPath); err == nil {
		const format = "Mon, 02 Jan 2006 15:04:05 GMT"
		req.Header.Set("If-Modified-Since", fi.ModTime().UTC().Format(format))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		log.Printf("%s: not modified", url)
		return nil
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %s", url, resp.Status)
	}

	log.Printf("downloading %s...\n", url)

	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("couldn't download %s: %w", url, err)
	}
	log.Printf("downloadIfNewer %s done", url)
	return nil
}

var versionFileContents = append([]byte(envoyVersion), byte('\n'))

func writeVersion(outputPath string) error {
	return os.WriteFile(outputPath, versionFileContents, 0o644)
}
