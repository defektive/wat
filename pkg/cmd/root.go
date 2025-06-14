package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/defektive/wat/pkg/wat"
	"net/netip"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var privateKey []byte
var serverPublicKey []byte
var endpointIP = "tring"
var endpointPort int
var tunnelLocalAddr netip.Addr
var tunnelDNSServerAddr netip.Addr

var wireGuardPeer *wat.Peer
var logLevel int

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wat",
	Short: "WireGuard Application Tunneller",
	Long:  `Userspace tunnels with WireGuard`,
	Example: `
Chain multiple tunnels

	./wat proxy -L 127.0.0.1:8080:10.4.0.1:8888 -L 3333:10.4.0.1:22 -R 8080:10.28.0.5:8080

`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		wireGuardPeer = wat.NewPeer(
			privateKey,
			serverPublicKey,
			endpointIP,
			[]netip.Addr{tunnelLocalAddr},
			[]netip.Addr{tunnelDNSServerAddr},
			logLevel,
		)

		if endpointPort != wireGuardPeer.Port {
			wireGuardPeer.Port = endpointPort
		}
	},
}

func mustDecode(base64str string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(base64str)
	if err != nil {
		panic(err)
	}
	return decoded

}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(privateKeyB64, serverPublicKeyB64, serverAddress, tunnelLocalIP, tunnelDNS string) {
	var err error

	if privateKeyB64 != "" {
		privateKey = mustDecode(privateKeyB64)
	}

	if serverPublicKeyB64 != "" {
		serverPublicKey = mustDecode(serverPublicKeyB64)
	}

	if serverAddress != "" {
		parsed, err := url.Parse(fmt.Sprintf("wg://%s", serverAddress))
		if err != nil {
			panic(err)
		}

		port, err := strconv.Atoi(parsed.Port())
		if err != nil {
			panic(err)
		}
		endpointIP = parsed.Hostname()
		endpointPort = port
	}

	if tunnelLocalIP != "" {
		tunnelLocalAddr = netip.MustParseAddr(tunnelLocalIP)
	}

	if tunnelDNS != "" {
		tunnelDNSServerAddr = netip.MustParseAddr(tunnelDNS)
	}

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	//rootCmd.Flags().StringVarP(&endpointIP, "endpoint-ip", "e", "", "Endpoint IP")
	rootCmd.PersistentFlags().IntVar(&logLevel, "log-level", 0, "Log Level")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
