package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

// A helper function to measure execution time
func timer(name string) func() {
	start := time.Now()
	return func() { fmt.Printf("%s %v\n", name, time.Since(start)) }
}
func confirmWithUser() bool {
	fmt.Print("\n Are you sure you want to continue? (yes/no) ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		return true
	} else {
		return false
	}
}
func main() {
	// Open the log file
	logFile, err := os.OpenFile("chown-pb.log", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	// Create a buffered writer for the log file
	bufferedLog := bufio.NewWriter(logFile)
	defer bufferedLog.Flush()

	// Set the log output to the buffered writer
	log.SetOutput(bufferedLog)

	//Use Mutex
	var mu sync.Mutex

	// Load the config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// Get the values from the config file
	dryRun := viper.GetBool("dryrun")     // dry run mode: only count no made any changes of UID`s
	log2File := viper.GetBool("log2file") // if true, write to logfile all changed files and folders.
	rootDir := viper.GetString("path")
	oldUID := viper.GetInt("olduid")
	newUID := viper.GetInt("newuid")
	includeFolders := viper.GetBool("includefolders") // Change UID of folders
	includeFiles := viper.GetBool("includefiles")     // Change UID of files

	// Print a message to the console to indicate that the program has started
	fmt.Println("\n Starting...\n")
	//Print selected config values to user and wait for confirmation
	if dryRun {
		fmt.Printf("[DRY-RUN] NO changing UID of folders and files! \n")

	} else {
		fmt.Printf("[WARNING] chown changing mode running, review you config and type `yes` to continue\n")
	}
	fmt.Println("\n -==The following config will be used-== \n")

	fmt.Println("path:           ", rootDir)
	fmt.Println("dryrun:         ", dryRun)
	fmt.Println("olduid:         ", oldUID)
	fmt.Println("newuid:         ", newUID)
	fmt.Println("includefolders: ", includeFolders)
	fmt.Println("includefiles:   ", includeFiles)
	fmt.Println("log2file:       ", log2File)

	// Call confirmWithUser to prompt the user
	if !confirmWithUser() {
		fmt.Println("\nExecution cancelled.")
		return
	}

	fmt.Println("\nExecution continued.")
	// Start a timer to measure the total execution time of the program
	defer timer("Total execution time:  ")()

	// Read the files and folders that need to have their UID changed
	var filesToChange []string
	var foldersToChange []string
	var countFiles, countFolders int

	err = filepath.WalkDir(rootDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		// Skip processing the root directory
		if path == rootDir {
			return nil
		}
		// Find all files and folders that need to have their UID changed
		if info.Sys().(*syscall.Stat_t).Uid == uint32(oldUID) {
			if info.IsDir() && includeFolders {
				foldersToChange = append(foldersToChange, path)

			} else if info.Mode().IsRegular() && includeFiles {
				filesToChange = append(filesToChange, path)

			}
			countFolders = len(foldersToChange)
			countFiles = len(filesToChange)
			mu.Lock()
			fmt.Printf("\rFolders: %d Files: %d", countFolders, countFiles)
			mu.Unlock()
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to read root directory: %v", err)
	}

	fmt.Println()

	// Change the UID of the folders
	for _, folder := range foldersToChange {
		if dryRun {

			mu.Lock()
			countFolders--
			fmt.Printf("\rFolders precessing remain: %d", countFolders)
			mu.Unlock()

		} else {
			err := os.Chown(folder, newUID, -1)
			if err != nil {
				log.Fatalf("Failed to change UID of folder %s: %v", folder, err)
			}
			if log2File {
				log.Printf("Changed UID of folder %s to %d\n", folder, newUID)
			}
			mu.Lock()
			countFolders--
			fmt.Printf("\rFolders precessing remain: %d", countFolders)
			mu.Unlock()

		}
	}
	fmt.Println()
	// Change the UID of the files
	for _, file := range filesToChange {

		if dryRun {

			mu.Lock()
			countFiles--
			fmt.Printf("\rFiles precessing remain: %d", countFiles)
			mu.Unlock()
		} else {
			err := os.Chown(file, newUID, -1)
			if err != nil {
				log.Fatalf("Failed to change UID of file %s: %v", file, err)
			}
			if log2File {
				log.Printf("Changed UID of file %s to %d\n", file, newUID)
			}
			mu.Lock()
			countFiles--
			fmt.Printf("\rFiles precessing remain: %d", countFiles)
			mu.Unlock()
		}

	}
	fmt.Println()
}
