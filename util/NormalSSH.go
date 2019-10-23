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
	KeyPath   = "/Users/jiashiran/go/src/jumpserver-automation/build/key/"
	//ips = []string{"39.105.202.8","39.107.243.195","39.96.50.197","123.56.17.69","47.95.241.44","47.94.130.119","47.93.218.38","101.200.52.2"}
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

type Input struct {
	Comman chan string
}

func (input Input) Read(p []byte) (n int, err error) {
	log.Println("wait read...")
	str, isOpen := <-input.Comman
	if !isOpen {
		return 0, io.EOF
	}
	if strings.Index(str, "\n") <= 0 {
		str = str + "\n"
	}
	log.Println("receive command:", str)
	if str == io.EOF.Error() {
		return 0, io.EOF
	}
	if str == "" {
		return 0, nil
	}
	bytes := []byte(str)
	for i, b := range bytes {
		p[i] = b
	}
	return len(bytes), nil
}

type Output struct {
	Result chan string
}

func (output Output) Write(p []byte) (n int, err error) {
	output.Result <- string(p)
	return len(p), nil
}

func NewSessionWithChan(client *ssh.Client, wsSesion *session.WsSesion) (*ssh.Session, chan string, chan string, func(duration time.Duration)) {
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	session.Setenv("LANG", "zh_CN.UTF-8")
	var in Input = Input{make(chan string)}
	session.Stdin = in
	var out Output = Output{make(chan string, 100)}
	session.Stdout = out
	go func(s *ssh.Session) {
		err = s.Shell()
		CheckErr(err, "session shell")
		err = s.Wait()
		CheckErr(err, "session wait")
		log.Println("session over")
	}(session)
	echo := func(duration time.Duration) {
		for {
			select {
			case msg, isOpen := <-out.Result:
				if isOpen {
					wsSesion.C.Emit("chat", msg)
					//log.Println(msg)
					if msg == "close channel session" {
						goto CLOSE
					}
					break
				} else {
					goto CLOSE
				}
			case <-time.After(duration):
				goto CLOSE
			}
		}
	CLOSE:
		log.Println("close channel session")
	}

	return session, in.Comman, out.Result, echo
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
