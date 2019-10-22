package util

import (
	"bufio"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"jumpserver-automation/session"
	"log"
	"net"
	"strings"
	"time"
)

var (
	ServerMap map[string]*ssh.Client
	KeyPath   = "build/key/"
	//ips = []string{""}
)

type RemoteOutPut struct {
	Name string
}

func (s *RemoteOutPut) Write(p []byte) (n int, err error) {
	log.Printf("remote.out:", string(p))
	return len(p), nil
}

type SSHServer struct {
	Name   string
	Config SSHConfig
}

/**
SSH配置
*/
type SSHConfig struct {
	User     string
	Password string
	KeyPath  string
	Ip       string
	Port     string
}

/**
创建ssh客户端
*/
func GetSSHClient(conf *SSHConfig) (*ssh.Client, error) {
	/*if conf.Password == ""{
		return nil , errors.New("conf param null!")
	}*/
	var config *ssh.ClientConfig
	if "" == conf.Password {
		key, err := ioutil.ReadFile(conf.KeyPath)
		if err != nil {
			log.Println("GetSSHClient.err", err)
		}
		signer, err := ssh.ParsePrivateKey([]byte(key))
		if err != nil {
			log.Println("GetSSHClient.err", err)
		}
		config = &ssh.ClientConfig{
			User: conf.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}
	} else {
		config = &ssh.ClientConfig{
			User: conf.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(conf.Password),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}
	}
	if "0" == conf.Port {
		conf.Port = "22"
	}
	client, err := ssh.Dial("tcp", conf.Ip+":"+conf.Port, config)
	if err != nil {
		//panic("Failed to dial: " + err.Error())
		return nil, err
	}
	return client, nil
}

/**
执行脚本
*/
func ExecuteShellWithChan(client *ssh.Client, shell string, wsSesion *session.WsSesion) string {
	defer func() {
		if err := recover(); err != nil {
			log.Println("ExecuteShell err:", err, shell)
		}
	}()
	if shell == "" {
		log.Println("shell is nil")
		return "shell is nil"
	}
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()
	session.Setenv("LANG", "zh_CN.UTF-8")
	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b []byte
	//session.Stdout = os.Stdout

	if b, err = session.Output(shell); err != nil {
		//panic("Failed to run: " + err.Error() + "shell:" + shell)
		log.Println("Failed to run: ", err.Error(), "shell:", shell)
	}
	wsSesion.C.Emit("chat", string(b))
	fmt.Println(string(b))
	return string(b)
}
func ExecuteShell(client *ssh.Client, shell string) string {
	defer func() {
		if err := recover(); err != nil {
			log.Println("ExecuteShell err:", err, shell)
		}
	}()
	if shell == "" {
		log.Println("shell is nil")
		return "shell is nil"
	}
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()
	session.Setenv("LANG", "zh_CN.UTF-8")
	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b []byte
	//session.Stdout = os.Stdout

	if b, err = session.Output(shell); err != nil {
		//panic("Failed to run: " + err.Error() + "shell:" + shell)
		log.Println("Failed to run: ", err.Error(), "shell:", shell)
	}

	//fmt.Println(string(b))
	return string(b)
}

func ExecuteShellGo(client *ssh.Client, shell string) {
	if shell == "" {
		log.Println("shell is nil")
		return
	}
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()
	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	//var b bytes.Buffer
	//session.Stdout = &b

	out, err := session.StdoutPipe()
	if err != nil {
		log.Println("estart shell err:", err)
	}
	read := bufio.NewReader(out)
	session.Setenv("LANG", "zh_CN.UTF-8")
	session.Start(shell)
	start := time.Now().Second()
	for {
		line, err := read.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		log.Print(line)
		if (time.Now().Second() - start) >= 10 {
			break
		}
	}

}

/**
创建sftp
*/
func GetSftpClient(client *ssh.Client) *sftp.Client {
	sftp, err := sftp.NewClient(client)
	if err != nil {
		log.Println("GetSftp.error", err)
	}
	return sftp
}

func ExecuteBatch(shell string) map[string]string {
	result := make(map[string]string)
	for ip, client := range ServerMap {
		res := ExecuteShell(client, shell)
		log.Println(ip, " execute ", shell, ", result:", res)
		res = strings.ReplaceAll(res, "\n", "")
		result[ip] = res
	}
	return result
}

func ExecuteShellWithReturn(ip, shell string) map[string]string {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Execute err:", err, ip)
		}
	}()
	result := make(map[string]string)
	res := ExecuteShell(ServerMap[ip], shell)
	log.Println(ip, " execute ", shell, ", result:", res)
	result[ip] = res
	return result
}

func UploadBatch(localFile, remotePath string) {
	for ip, client := range ServerMap {
		UploadPath(client, localFile, remotePath)
		log.Println(ip, " uploaded file:", localFile, " to remotePath ", remotePath)
	}
}

func InitServer(ips []string) {
	ServerMap = make(map[string]*ssh.Client)
	for _, ip := range ips {
		client, err := GetSSHClient(&SSHConfig{
			User:     "root",
			Password: "",
			KeyPath:  "",
			Ip:       ip,
			Port:     "22",
		})
		if err != nil {
			log.Println(err)
		} else {
			ServerMap[ip] = client
		}
	}
}
