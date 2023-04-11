package container

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

var (
	RUNNING             string = "running"
	STOP                string = "stopped"
	Exit                string = "exited"
	DefaultInfoLocation string = "/var/run/mydocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
	RootUrl				string = "/root"
	MntUrl				string = "/root/mnt/%s"
	WriteLayerUrl 		string = "/root/writeLayer/%s"
)

type ContainerInfo struct {
	Pid         string `json:"pid"`        //容器的init进程在宿主机上的 PID
	Id          string `json:"id"`         //容器Id
	Name        string `json:"name"`       //容器名
	Command     string `json:"command"`    //容器内init运行命令
	CreatedTime string `json:"createTime"` //创建时间
	Status      string `json:"status"`     //容器的状态
	Volume      string `json:"volume"`     //容器的数据卷
	PortMapping []string `json:"portmapping"` //端口映射
}
/*
这里是父进程，也就是当前进程执行的内容，
1.这里的/process/self/exe调用中，/proc/self/指的是当前运行进程自己的环境。exec其实就是自己调用自己
使用这种方式对创建出来的进程进行初始化
2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化
进程的一下环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用namespace隔离新创建的进程和外部环境。
4. 如果用户指定了-ti参数，就需要把当前进程的输入输出导入到标准的输入输出上。
*/
func NewParentProcess(tty bool, containerName, volume, imageName string, envSlice []string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}
	initCmd, err := os.Readlink("/proc/self/exe")
	if err != nil {
		log.Errorf("get init process error %v", err)
		return nil, nil
	}

	cmd := exec.Command(initCmd, "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirURL, 0622); err != nil {
			log.Errorf("NewParentProcess mkdir %s error %v", dirURL, err)
			return nil, nil
		}
		stdLogFilePath := dirURL + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			log.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
	}

	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(os.Environ(), envSlice...)
	NewWorkSpace(volume, imageName, containerName)
	cmd.Dir = fmt.Sprintf(MntUrl, containerName)
	return cmd, writePipe
}

//在Go语言中，os.Pipe函数用于创建一个管道，该管道可以在同一个进程内的不同协程之间进行通信，
//也可以在不同进程之间进行通信。管道是一种特殊的文件描述符，它可以用于在进程之间进行双向通信，一个进程可以将数据写入管道，另一个进程可以从管道中读取数据。
func NewPipe() (*os.File, *os.File, error) {
	//该函数返回两个文件描述符，其中一个用于读取管道中的数据，另一个用于写入数据到管道中。
	//在函数执行过程中，会创建一个匿名管道，并返回两个文件描述符，它们分别对应管道的读端和写端。可以通过这两个文件描述符在进程之间进行数据传输。
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
