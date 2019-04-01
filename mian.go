package main

import (
	"jumpserver-automation/ws"
)

func main() {

	//util.OperatLb("LB aliyun slb in i-uf68a1l5ulumn2tizgw5 lb-uf6lfi06q95h5bcyxb87e 8089")
	//util.OperatLb("LB aws alb in i-0e41029513647e310 arn:aws-cn:elasticloadbalancing:cn-north-1:147022339119:targetgroup/sip-sbc-302/1dfa6510b0676035 8089")
	//util.OperatLb("LB aws elb in i-00722409683af6fab lbaaa 8089")

	service()
}

func service() {
	ws.Service()
}

/*func WaitStop()  {
	// Go signal notification works by sending `os.Signal`
	// values on a channel. We'll create a channel to
	// receive these notifications (we'll also make one to
	// notify us when the program can exit).
	sigs := make(chan os.Signal)
	done := make(chan bool, 1)
	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGILL, syscall.SIGTRAP, syscall.SIGABRT, syscall.SIGBUS, syscall.SIGFPE, syscall.SIGKILL,syscall.SIGSEGV,syscall.SIGPIPE,syscall.SIGALRM,syscall.SIGTERM)
	// This goroutine executes a blocking receive for
	// signals. When it gets one it'll print it out
	// and then notify the program that it can finish.
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		store.Close()
		done <- true
	}()
	// The program will wait here until it gets the
	// expected signal (as indicated by the goroutine
	// above sending a value on `done`) and then exit.
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}*/
