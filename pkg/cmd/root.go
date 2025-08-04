package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/defektive/wat/pkg/wat"
	"github.com/spf13/cobra"
	"net/netip"
	"net/url"
	"os"
	"strconv"
)

var defaultPrivateKeyB64 = ""
var defaultServerPublicKeyB64 = ""
var defaultServerAddress = ""
var defaultTunnelLocalAddress = ""
var defaultTunnelDNSServer = ""
var defaultDynamicTunnels = ""
var defaultLocalTunnels = ""
var defaultRemoteTunnels = ""

var privateKey []byte
var serverPublicKey []byte
var endpointIP = ""
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
		privateKey = mustDecode(defaultPrivateKeyB64)
		serverPublicKey = mustDecode(defaultServerPublicKeyB64)
		parsed, err := url.Parse(fmt.Sprintf("wg://%s", defaultServerAddress))
		if err != nil {
			panic(err)
		}

		port, err := strconv.Atoi(parsed.Port())
		if err != nil {
			// no port specified
			port = wat.DefaultPort
		}
		endpointIP = parsed.Hostname()
		endpointPort = port
		tunnelLocalAddr = netip.MustParseAddr(defaultTunnelLocalAddress)
		tunnelDNSServerAddr = netip.MustParseAddr(defaultTunnelDNSServer)

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
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVarP(&defaultPrivateKeyB64, "private-key", "K", defaultPrivateKeyB64, "Base64 private key")
	rootCmd.PersistentFlags().StringVarP(&defaultTunnelLocalAddress, "tunnel-ip", "I", defaultTunnelLocalAddress, "IP address for wireguard interface")
	rootCmd.PersistentFlags().StringVarP(&defaultServerAddress, "server", "S", defaultServerAddress, "Server to connect to")
	rootCmd.PersistentFlags().StringVarP(&defaultServerPublicKeyB64, "server-key", "P", defaultServerPublicKeyB64, "Base64 public key for server")
	rootCmd.PersistentFlags().StringVarP(&defaultTunnelDNSServer, "tunnel-dns-server", "d", defaultTunnelDNSServer, "DNS server to use for wireguard interface")
	rootCmd.PersistentFlags().IntVar(&logLevel, "log-level", 0, "Log Level")

	// required flags dont work with default values...
	// if compiled without these, make them required here
	if defaultPrivateKeyB64 == "" {
		rootCmd.MarkPersistentFlagRequired("private-key")
	}

	if defaultServerAddress == "" {
		rootCmd.MarkPersistentFlagRequired("server")
	}
	if defaultServerPublicKeyB64 == "" {
		rootCmd.MarkPersistentFlagRequired("server-key")
	}

	if defaultTunnelLocalAddress == "" {
		rootCmd.MarkPersistentFlagRequired("tunnel-ip")
	}

	if defaultTunnelDNSServer == "" {
		rootCmd.MarkPersistentFlagRequired("tunnel-dns-server")
	}

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
