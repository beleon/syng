package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var syncAfter = 30000 //Time in milliseconds
var forceSyncAfter = 10 * syncAfter //Time in milliseconds
var mutex = &sync.Mutex{}

func main() {
	if !exists(".git") {
		println("not in git root directory (.git directory not found)")
		os.Exit(1)
	}

	pull()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for _ = range c {
			println("received SIGINT/SIGTERM, saving changes and stopping...")
			update()
			os.Exit(1)
		}
	}()

	if loadIntEnv("SYNG_SYNC_AFTER", &syncAfter) {
		fmt.Printf("sync after env set: syncing after %d milliseconds\n", syncAfter)
	}

	if loadIntEnv("SYNG_FORCE_SYNC_AFTER", &syncAfter) {
		fmt.Printf("force sync after env set: syncing after %d milliseconds\n", syncAfter)
	}

	lastSync := time.Now()
	clean := true
	syncAfterDuration := time.Duration(syncAfter) * time.Millisecond
	autoDeletedDuration := syncAfterDuration + 1 * time.Millisecond
	forceSyncAfterDuration := time.Duration(forceSyncAfter) * time.Millisecond
	println("Watching for changes...")
	for {
		now := time.Now()
		if !clean && now.Sub(lastSync) > forceSyncAfterDuration {
			update()
			clean = true
			time.Sleep(syncAfterDuration)
			continue
		}
		cf := changedFiles()
		if len(cf) > 0 {
			shortest := forceSyncAfterDuration + 1 * time.Millisecond
			for _, file := range changedFiles() {
				lm, deleted := lastModified(file)
				modDur := now.Sub(lm)
				if !deleted && modDur < shortest {
					shortest = modDur
				} else if deleted && autoDeletedDuration < shortest {
					shortest = autoDeletedDuration
				}
			}
			if clean {
				lastSync = now.Add(-shortest)
			}
			clean = false
			if shortest > syncAfterDuration {
				update()
				clean = true
				time.Sleep(syncAfterDuration)
				continue
			}
			untilForceSync := forceSyncAfterDuration - now.Sub(lastSync)
			untilNormalSync := syncAfterDuration - shortest
			if untilForceSync < untilNormalSync {
				time.Sleep(untilForceSync)
			} else {
				time.Sleep(untilNormalSync)
			}
			continue
		}
		time.Sleep(syncAfterDuration)
	}
}

func pull() {
	println("pulling updates")
	pullCmd := exec.Command("git", "pull")
	pullCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr

	err := pullCmd.Run()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}

func update() {
	mutex.Lock()
	defer mutex.Unlock()
	println("staging changes...")
	addCmd := exec.Command("git", "add", ".")
	addCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr

	err := addCmd.Run()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	println("creating commit...")
	commitCmd := exec.Command("git", "commit", "-m", "Update")
	commitCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr

	err = commitCmd.Run()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	println("pushing...")
	pushCmd := exec.Command("git", "push")
	pushCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr

	err = pushCmd.Run()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil { return true }
	if os.IsNotExist(err) { return false }
	fmt.Printf("%s\n", err.Error())
	os.Exit(1)
	return false
}

func changedFiles() []string {
	out, err := exec.Command("git", "status", "-uall", "--porcelain").Output()
	if err != nil {
		return make([]string, 0)
	}
	res := make([]string, 0)
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		file := strings.TrimLeft(line, "\t \n")
		res = append(res, file[strings.Index(file, " ")+1:])
	}
	return res
}

//Return modification time and whether or not the file was delete
func lastModified(path string) (time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Unix(0, 0), true
		}
		fmt.Printf("error checking modtime of file %s: %s\n", path, err.Error())
		return time.Now(), false
	}
	return info.ModTime(), false
}

func loadIntEnv(key string, target *int) bool {
	val := os.Getenv(key)
	if val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			fmt.Printf("erroring interpreting %s (%s) as int: %s\n", key, val, err.Error())
		} else {
			*target = i
			return true
		}
	}
	return false
}