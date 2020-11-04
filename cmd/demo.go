package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// MonitoredCmd is a wrapper around exec.Cmd
// the signalChannel will
type MonitoredCmd struct {
	sigChan chan os.Signal
	done    chan bool
	*exec.Cmd
}

// NewMonitoredCmd creates a new MonitoredCmd to monitor a signal channel.
func NewMonitoredCmd(c chan os.Signal, cmd *exec.Cmd) *MonitoredCmd {
	return &MonitoredCmd{
		sigChan: c,
		Cmd:     cmd,
		done:    make(chan bool, 1),
	}
}

func (grp *MonitoredCmd) Run() error {
	fmt.Println("running command")

	grp.Cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	if err := grp.Cmd.Start(); err != nil {
		return err
	}
	defer func() {
		grp.done <- true
	}()
	go func() {
		select {
		case <-grp.sigChan:
			syscall.Kill(-grp.Cmd.Process.Pid, syscall.SIGINT)
			fmt.Println("after interrupt killed")
			return
		case <-grp.done:
			return
		}
	}()
	err := grp.Cmd.Wait()
	fmt.Println("wait output/err", err)
	return err
}

// demoCmd represents the demo command
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		osCmd := exec.Command("zsh", "-c", "go run dev-services/ticker/ticker.go")
		osCmd2 := exec.Command("zsh", "-c", "go run dev-services/ticker/ticker.go")

		// set stdout and err to stdout and err
		osCmd.Stdout = os.Stdout
		osCmd.Stderr = os.Stdout
		osCmd2.Stdout = os.Stdout
		osCmd2.Stderr = os.Stdout

		// what directory to run this command in
		osCmd.Dir = "/Users/chao/go/src/github.com/alexchao26/oneterminal"
		osCmd2.Dir = "/Users/chao/go/src/github.com/alexchao26/oneterminal"

		sigChan := make(chan os.Signal, 1)
		monitored := NewMonitoredCmd(sigChan, osCmd)

		sigChan2 := make(chan os.Signal, 1)
		monitored2 := NewMonitoredCmd(sigChan2, osCmd2)

		go monitored.Run()
		go monitored2.Run()

		// make a channel that will amplify termination signals to all cancel channels?
		amplifyChannel := make(chan os.Signal, 1)
		signal.Notify(amplifyChannel, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

		go func() {
			sig := <-amplifyChannel
			fmt.Println("received signal", sig)
			sigChan <- sig
			sigChan2 <- sig

			// the signal needs time to kill the other process before exiting this/main app...
			time.Sleep(time.Second * 2)
			os.Exit(1)
		}()
		// defer func() {
		// 	fmt.Println("sending interrupt signal")
		// 	sigChan <- syscall.SIGINT
		// }()

		// 5 second auto kill
		// fmt.Println("OS CMD", osCmd)
		// time.Sleep(time.Second * 5)
		// fmt.Println("sending interrupt signal")
		// sigChan <- syscall.SIGINT

		time.Sleep(time.Second * 20)
		fmt.Println("Exiting...")
		amplifyChannel <- syscall.SIGINT
		time.Sleep(time.Second * 5)
		//
	},
}

func init() {
	rootCmd.AddCommand(demoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// demoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// demoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
