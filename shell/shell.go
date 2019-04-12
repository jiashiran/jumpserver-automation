package shell

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
)

var OUT = make(chan string, 100)
var IN = make(chan string)

type JumpserverConfig struct {
	User     string
	Password string
	Ip       string
	Port     int
}

func NewJumpserverClient(conf *JumpserverConfig) (*ssh.Client, error) {
	var config ssh.ClientConfig
	var authMethods []ssh.AuthMethod
	authMethods = append(authMethods, ssh.Password(conf.Password))
	authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, 0, len(questions))
		for i, q := range questions {
			fmt.Print(q)
			if echos[i] {
				scan := bufio.NewScanner(os.Stdin)
				if scan.Scan() {
					answers = append(answers, scan.Text())
				}
				err := scan.Err()
				if err != nil {
					return nil, err
				}

			} else {
				b, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return nil, err
				}
				fmt.Println()
				answers = append(answers, string(b))
			}
		}
		return answers, nil
	}))
	config = ssh.ClientConfig{
		User: conf.User,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", conf.Ip+":"+strconv.Itoa(conf.Port), &config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
		return nil, err
	}

	return client, nil
}

func NewShell(client *ssh.Client) *ssh.Session {
	session, err := client.NewSession()
	CheckErr(err, "create new session")
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Setenv("LANG", "zh_CN.UTF-8")
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = session.RequestPty("xterm", 100, 200, modes)
	if err != nil {
		log.Println(errors.New("unable request pty  " + err.Error()))
	}

	go func(s *ssh.Session) {
		err = s.Shell()
		CheckErr(err, "session shell")
		err = s.Wait()
		CheckErr(err, "session wait")
		log.Println("session over")
	}(session)
	return session
}

func Shell() {
	client, err := NewJumpserverClient(&JumpserverConfig{
		User:     "",
		Password: "",
		Ip:       "",
		Port:     0,
	})
	if err != nil {
		log.Fatal("gt client err:", err)
	}
	NewShell(client)
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

func GetSftp(client *ssh.Client) *sftp.Client {
	sftp, err := sftp.NewClient(client)
	if err != nil {
		log.Println("GetSftp.error", err)
	}
	return sftp
}

const Separator = "/"

/**
上传目录
*/
func UploadPath(client *ssh.Client, localPath, remotePath string) {
	sftp := GetSftp(client)
	defer sftp.Close()
	uploadPath(localPath, sftp, remotePath)
}

/**
上传目录子方法
*/
func uploadPath(file string, sftp *sftp.Client, remotePath string) {
	inputFile, inputError := os.Open(file)
	if inputError != nil {
		log.Println(os.Stderr, "File Error: %s\n", inputError)
	}

	fileInfo, err := inputFile.Stat()
	if err != nil {
		log.Println("fileinfo err:", err)
	}
	defer inputFile.Close()

	if fileInfo.IsDir() {
		//mkdir
		path := remotePath + Separator + fileInfo.Name()
		sftp.Mkdir(path)
		//log.Println(path)

		fileInfo, err := inputFile.Readdir(-1)
		if err == nil {
			for _, f := range fileInfo {

				uploadPath(file+Separator+f.Name(), sftp, path)

			}
		}

	} else {
		//copy file
		//log.Println(remotePath + Separator + fileInfo.Name())
		uploadFile(sftp, file, remotePath+Separator+fileInfo.Name())
	}

}

/**
上传文件
*/
func uploadFile(sftp *sftp.Client, localFile, remotePath string) {
	log.Println(localFile, ",", remotePath)
	// leave your mark
	inputFile, inputError := os.Open(localFile)
	//fileInfo , err := inputFile.Stat();
	defer inputFile.Close()

	f, err := sftp.Create(remotePath)

	if err != nil {
		log.Println("sftp.Create.err", err)
	}

	if inputError != nil {
		log.Println("File Error: %s\n", inputError)
	}

	fileReader := bufio.NewReader(inputFile)
	counter := 0
	for {
		buf := make([]byte, 20480)
		n, err := fileReader.Read(buf)
		if err == io.EOF {
			break
		}
		counter++
		//fmt.Printf("%d,%s", n, string(buf))
		if n == 0 {
			break
		}
		//fmt.Println(string(buf))
		if _, err := f.Write(buf[0:n]); err != nil {
			log.Println(err)
		}

	}
	// check it's there
	fi, err := sftp.Lstat(remotePath)
	if err != nil {
		log.Println("sftp.Lstat.error", err)
	}
	log.Println(fi)

}

/**
删除文件
*/
func RemoveFile(remoateFile string, sftp *sftp.Client) {
	err := sftp.Remove(remoateFile)
	if err != nil {
		log.Println(err)
	}
}

/**
查看文件列表
*/
func ListPath(sftp *sftp.Client, remotePath string) {
	//defer sftp.Close()
	// walk a directory
	w := sftp.Walk(remotePath)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		log.Println(w.Path())
	}
}

func CheckErr(err error, msg string) {
	if err != nil {
		log.Println(msg+" err:", err)
	}
}
