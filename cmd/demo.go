package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/alexchao26/oneterminal/pkg/monitor-cmd"

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
		// osCmd := exec.Command("zsh", "-c", "go run dev-services/ticker/ticker.go")
		// osCmd2 := exec.Command("zsh", "-c", "go run dev-services/ticker/ticker.go")

		coordinator := monitor.NewCoordinator()

		m1 := monitor.NewMonitoredCmd(
			"go run dev-services/ticker/ticker.go",
			coordinator,
			monitor.SetCmdDir("/Users/chao/go/src/github.com/alexchao26/oneterminal"))

		go m1.Run()

		// make a channel that will relay termination signals to all cancel channels?
		relayChannel := make(chan os.Signal, 1)
		signal.Notify(relayChannel, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

		for {
			select {
			case sig := <-relayChannel:
				fmt.Println("received termination signal to relay", sig)
				m1.Interrupt()

				// the signal needs time to kill the other process before exiting this/main app...
				// time.Sleep(time.Second * 2)
				// os.Exit(1)
			case <-coordinator.SyncChan:
				fmt.Println("underlying command done")
				coordinator.FinishCommand()
				if coordinator.GetStatus() == monitor.StatusDone {
					fmt.Println("coordinator done... exiting...")
					os.Exit(1)
				}
				fmt.Println("still pending jobs...")
			}
		}

		// time.Sleep(time.Second * 20)
		// fmt.Println("Exiting...")
		// relayChannel <- syscall.SIGINT

		// things go bad if the main go routine ends before everything else shuts down
		// time.Sleep(time.Second * 45)
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
