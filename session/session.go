package session

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"strings"
)

type WsSesion struct {
	ID      string
	Client  *ssh.Client
	Session *JumpserverSession
	OUT    chan string//           = make(chan string, 100)
	IN     chan string//           = make(chan string)
	LoginServer bool
}

type JumpserverSession struct {
	*ssh.Session
	In  *Input
	Out *Output
	Health bool
	CheckURL string
	WebSesion *WsSesion
	CheckCount	int
	CheckCommand string
}
func (s *JumpserverSession) SendCommand(command string) {
	s.In.In <- command
	log.Println("send command:", command)
}


type Input struct {
	In chan string
}

func (in *Input) Read(p []byte) (n int, err error) {
	log.Println("wait read...")
	str := <-in.In
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
	Out chan string
	JumpserverSession *JumpserverSession
}

func (out *Output) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		log.Println("session close")
		return -1, io.EOF
	}
	output := string(p)
	log.Println(out)
	log.Println(out.JumpserverSession)
	log.Println(out.JumpserverSession.WebSesion)
	if strings.Contains(output, "Opt>") {
		out.JumpserverSession.WebSesion.LoginServer = false
	}
	if out.JumpserverSession.WebSesion.LoginServer == false && (strings.Contains(output, "$") || strings.Contains(output, "#")) {
		out.JumpserverSession.WebSesion.LoginServer = true
	}
	if out.JumpserverSession.CheckURL != "" && strings.Contains(output,out.JumpserverSession.CheckURL) && !strings.Contains(output,out.JumpserverSession.CheckCommand){
		out.JumpserverSession.CheckCount += 1
		if out.JumpserverSession.CheckCount >= 3 {
			out.JumpserverSession.Health = true
		}
	}
	out.JumpserverSession.WebSesion.OUT <- output
	//log.Println("output:", output)
	return len(p), nil
}


