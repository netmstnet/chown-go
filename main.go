package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

// job represents a file or folder whose UID needs to be changed
type job struct {
	path string
}

// changeUID changes the UID of a file or folder
func changeUID(j job, oldUID, newUID int) error {
	// Check if the file or folder has the old UID
	info, err := os.Stat(j.path)
	if err != nil {
		return fmt.Errorf("failed to get info for %s: %w", j.path, err)
	}
	if info.Sys().(*syscall.Stat_t).Uid != uint32(oldUID) {
		if log2File {
			log.Printf("Skipping %s because it does not have the old UID\n", j.path)
		}
		return nil
	}

	// Change the UID of the file or folder
	if err := os.Chown(j.path, newUID, -1); err != nil {
		return fmt.Errorf("failed to change UID of %s: %w", j.path, err)
	}
	if log2File {
		log.Printf("Changed UID of %s to %d\n", j.path, newUID)
	}
	return nil
}

// createJobs creates a channel of jobs from the given slice of file and folder paths
func createJobs(paths []string) <-chan job {
	jobs := make(chan job, len(paths))
	for _, path := range paths {
		jobs <- job{path: path}
	}
	close(jobs)
	return jobs
}

// createWorkers starts a pool of worker goroutines and returns a channel to collect results
func createWorkers(numWorkers int, jobs <-chan job, oldUID, newUID int) <-chan bool {
	results := make(chan bool, len(jobs))
	for w := 1; w <= numWorkers; w++ {
		go func() {
			for j := range jobs {
				if err := changeUID(j, oldUID, newUID); err != nil {
					log.Fatal(err)
				}
				results <- true
			}
		}()
	}
	return results
}

// getFilesAndFolders returns separate slices of file and folder paths found in the given directory
func getFilesAndFolders(path string) ([]string, []string, error) {
	var filesToChange []string
	var foldersToChange []string
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			foldersToChange = append(foldersToChange, path)
		} else {
			filesToChange = append(filesToChange, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error counting files and folders: %w", err)
	}
	return filesToChange, foldersToChange, nil
}

var (
	newUID    int  = 1000
	oldUID    int  = 0
	log2File  bool = true
	dryRun    bool = true
	numWorkers int = 10
)

func main() {

	path := "/tmp/files"

	filesToChange, foldersToChange, err := getFilesAndFolders(path)
	if err != nil {
		log.Fatal(err)
	}

	// If dryRun is true, print the number of files and folders to change and return
	if dryRun {
		fmt.Printf("Found %d files and %d folders to change in
