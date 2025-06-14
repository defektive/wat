package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var ProxyListener = ":8080"

var LocalTunnels = []string{}
var RemoteTunnels = []string{}

// proxyCmd represents the base command when called without any subcommands
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "proxy",
	Long:  `Start the proxy server`,
	Example: `
Expose port 3306 on localhost and tunnel all traffic to port 3306 on a host on the wireguard network.

	./wat proxy -L 127.0.0.1:3306:10.4.0.1:3306

Expose port 2222 on broadcast and tunnel all traffic to port 22 on a host on the wireguard network.

	./wat proxy -L 2222:10.4.0.1:22

Accept traffic from wireguard network on port 443, forward it to port 443 on the local network.

	./wat proxy -L 443:intranet.example.com:443
`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("proxy called", ProxyListener)

		go func() {
			if err := wireGuardPeer.LocalTunnels(LocalTunnels...); err != nil {
				log.Println("[p] failed to setup local tunnels", err)
			}
		}()

		if err := wireGuardPeer.RemoteTunnels(RemoteTunnels...); err != nil {
			log.Println("[p] failed to setup remote tunnels", err)
		}
	},
}

func init() {

	proxyCmd.Flags().StringSliceVarP(&LocalTunnels, "local", "L", LocalTunnels, "Local tunnels. 80:remote-wg-addr:8080 will expose port 80 on this machine to port 8080 on a machine in the wireguard network ")
	proxyCmd.Flags().StringSliceVarP(&RemoteTunnels, "remote", "R", RemoteTunnels, "remote tunnels 443:100.23.23.12:443 will expose port 443 on this host's wireguard interface to port 443 on a machine in this machines network")
	rootCmd.AddCommand(proxyCmd)
}
