package subsystems

import (
	"fmt"
	"strings"
	"os"
	"path"
	"bufio"
)

//mountinfo可以找出与当前进程相关的mount信息。
//Cgroups的hierarchy的虚拟文件系统  是通过 cgroup类型文件系统的mount挂载上去的。
//option中加上subsystem， 代表挂载的subsystem类型。 这样就可以在mountinfo中找到对应的subsystem的挂载目录了。
func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				return fields[4]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

//GetCgroupPath函数 找到对应subsystem挂载的hierarchy相对路径 对应的 cgroup在虚拟文件系统中的路径
//通过该目录的读写  去操作cgroup

//如何找到挂载了subsystem的hierarchy的挂载目录的呢？   
func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsystem)
	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {
			} else {
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		return path.Join(cgroupRoot, cgroupPath), nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}