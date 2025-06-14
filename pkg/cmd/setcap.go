package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// setcapCmd represents the base command when called without any subcommands
var setcapCmd = &cobra.Command{
	Use:     "setcap",
	Short:   "set capabilities to open low ports",
	Long:    `set capabilities to open low ports`,
	Example: `sudo wat setcap`,
	Run: func(cmd *cobra.Command, args []string) {
		me, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("found executable", me)
		setcap := "setcap"
		setcapPath, err := exec.LookPath(setcap)
		if err != nil {
			log.Fatal(err)
		}
		cap := "cap_net_bind_service=+ep"
		log.Println("calling setcap", fmt.Sprintf("'%s'", cap), me)

		if err := syscall.Exec(setcapPath, []string{setcap, cap, me}, []string{}); err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(setcapCmd)
}
