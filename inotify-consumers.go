package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type Consumer struct {
	pid     string
	ruid    string
	wdcount int64
}

var inotify_consumers []Consumer

type mountinfo struct {
	path string
	Dev  uint64
	Ino  uint64
}

func (info *mountinfo) Dump() {
	fmt.Printf("%s\tDev:%d\tIno:%d\n", info.path, info.Dev, info.Ino)
}

func read_mount_info() []mountinfo {
	path := "/proc/mounts"
	fd, err := os.Open(path)
	if err != nil {
		log.Fatalf("read file %v error: %v\n", path, err)
	}
	defer fd.Close()
	reader := bufio.NewReader(fd)
	line, err := reader.ReadString('\n')
	var infos []mountinfo
	for len(line) > 0 {
		if err != nil {
			break
		}
		dir := regexp.MustCompile("[[:space:]]+").Split(line, 3)[1]
		if strings.HasPrefix(dir, "/") {
			var info mountinfo
			stat, err := getinode(dir)
			if err == nil {
				info.path = dir
				info.Dev = stat.Dev
				info.Ino = stat.Ino
			}
			infos = append(infos, info)
		}

		line, err = reader.ReadString('\n')
	}
	return infos
}

func getinode(path string) (syscall.Stat_t, error) {
	var stat syscall.Stat_t
	err := syscall.Lstat(path, &stat)
	if err != nil {
		return stat, err
	}
	return stat, nil
}

func get_ruid(pid string) string {
	path := fmt.Sprintf("/proc/%s/status", pid)
	fd, err := os.Open(path)
	if err != nil {
		//log.Fatalf("read file %v error: %v\n", path, err)
		return ""
	}
	defer fd.Close()
	reader := bufio.NewReader(fd)
	line, err := reader.ReadString('\n')
	var ruid string
	for len(line) > 0 {
		if err != nil {
			break
		}
		if strings.Contains(line, "Uid:") == false {
			line, err = reader.ReadString('\n')
			continue
		}
		ruid = string(regexp.MustCompile("[0-9]+").Find([]byte(line)))
		break

	}
	return ruid
}

func readfile(path string) string {
	fd, err := os.Open(path)
	if err != nil {
		//log.Fatalf("read file %v error: %v\n", path, err)
		return ""
	}
	defer fd.Close()
	buf := make([]byte, 4096)
	count, _ := fd.Read(buf)
	return string(string(buf[0:count]))
}

func print_info() {
	var total_wdcount int64
	fmt.Printf("%-10s %-10s %-10s %-40s %s\n", "COUNT", "PID", "UID", "COMM", "CMDLINE")
	for _, consumer := range inotify_consumers {
		total_wdcount += consumer.wdcount
		comm := readfile(fmt.Sprintf("/proc/%v/comm", consumer.pid))
		comm = strings.TrimRight(comm, "\n")
		cmdline := readfile(fmt.Sprintf("/proc/%v/cmdline", consumer.pid))
		fmt.Printf("%-10d %-10s %-10s %-40s %s\n", consumer.wdcount, consumer.pid, consumer.ruid, comm, cmdline)
	}
	//fmt.Println("Total inotify watches: ", total_wdcount)
}

func check_pid_inotify_info(pid string) {
	path := fmt.Sprintf("/proc/%v/fd", pid)
	files, err := os.ReadDir(path)
	if err != nil {
		//log.Fatal(err)
		return
	}
	var consumer Consumer
	consumer.pid = pid
	consumer.ruid = get_ruid(pid)

	buf := make([]byte, 50)
	for _, file := range files {
		//log.Println(file.Name())
		path := fmt.Sprintf("%s/%s", path, file.Name())
		size, err := syscall.Readlink(path, buf)
		if err != nil {
			//log.Fatalf("readline %s error: %v\n", path, err)
			continue
		}
		if size <= 0 {
			continue
		}
		link := string(buf[0:size])
		if link != "anon_inode:inotify" {
			continue
		}

		infofd, err := os.Open(fmt.Sprintf("/proc/%s/fdinfo/%s", pid, file.Name()))
		if err != nil {
			//log.Fatalln(err)
			continue
		}
		defer infofd.Close()
		reader := bufio.NewReader(infofd)
		line, err := reader.ReadString('\n')

		for len(line) > 0 {
			if err != nil {
				break
			}
			if strings.Contains(line, "inotify wd:") {
				consumer.wdcount += 1
			}
			line, err = reader.ReadString('\n')
		}
	}
	if consumer.wdcount > 0 {
		inotify_consumers = append(inotify_consumers, consumer)
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	files, err := os.ReadDir("/proc")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() == false {
			continue
		}
		pid_int, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}

		if pid_int == os.Getpid() {
			continue
		}
		check_pid_inotify_info(file.Name())
	}
	print_info()

	infos := read_mount_info()
	for _, info := range infos {
		info.Dump()
	}
}
