package main

import (
	"jumpserver-automation/ws"
)

func main() {

	ws.Service()

	//WaitStop()
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
