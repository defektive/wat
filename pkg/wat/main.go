package wat

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
	"io"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"
)

const DefaultPort = 51820
const DefaultMTU = 1420

type Peer struct {
	PrivateKey      []byte
	ServerPublicKey []byte
	Address         string
	Port            int
	LogLevel        int

	LocalAddresses []netip.Addr
	DNSServers     []netip.Addr
	MTU            int

	tunnelInst *tun.Device
	tunnelNet  *netstack.Net
	wgDevice   *device.Device
}

func (p *Peer) getTunnel() (*tun.Device, *netstack.Net, error) {

	if p.tunnelInst == nil {

		tunnelInst, tNet, err := netstack.CreateNetTUN(p.LocalAddresses, p.DNSServers, DefaultMTU)
		if err != nil {
			return nil, nil, err
		}

		p.tunnelInst = &tunnelInst
		p.tunnelNet = tNet
	}

	return p.tunnelInst, p.tunnelNet, nil
}

func (p *Peer) getDevice() (*device.Device, error) {
	if p.wgDevice == nil {

		tunnelInst, _, err := p.getTunnel()
		if err != nil {
			return nil, err
		}

		dev := device.NewDevice(*tunnelInst, conn.NewDefaultBind(), device.NewLogger(p.LogLevel, "[p] "))

		wgConf := bytes.NewBuffer(nil)
		_, err = fmt.Fprintf(
			wgConf,
			"private_key=%x\npublic_key=%x\nendpoint=%s:%d\nallowed_ip=%s\n",
			p.PrivateKey,
			p.ServerPublicKey,
			p.Address,
			p.Port,
			"0.0.0.0/0",
		)
		if err != nil {
			return nil, err
		}

		if err = dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
			return nil, err
		}

		p.wgDevice = dev
		go p.KeepAlive()
	}

	return p.wgDevice, nil
}

func (p *Peer) KeepAlive() {
	for {
		//p.wgDevice.SendKeepalivesToPeersWithCurrentKeypair()

		// ghetto hack to force connection so we can listen on it
		// todo: learn how to do this without ghetto hack
		p.tunnelNet.LookupHost("test.local")
		time.Sleep(60 * time.Second)
	}
}

func (p *Peer) Dial(proto, address string) (net.Conn, error) {
	_, err := p.getDevice()
	if err != nil {
		return nil, err
	}

	return p.tunnelNet.Dial(proto, address)
}

func (p *Peer) RemoteTunnels(tunnels ...string) error {

	var waitGroup sync.WaitGroup

	for _, tunnel := range tunnels {
		protocol := "tcp"
		localAddr := ""
		remoteAddr := ""
		tunnelDef := tunnel
		protoSlice := strings.Split(tunnel, "/")
		if len(protoSlice) == 2 {
			protocol = protoSlice[0]
			tunnelDef = protoSlice[1]
		}

		tunnelSlice := strings.Split(tunnelDef, ":")
		if len(tunnelSlice) == 3 {
			localAddr = fmt.Sprintf(":%s", tunnelSlice[0])
			remoteAddr = fmt.Sprintf("%s:%s", tunnelSlice[1], tunnelSlice[2])
		}
		if len(tunnelSlice) == 4 {
			localAddr = fmt.Sprintf("%s:%s", tunnelSlice[0], tunnelSlice[1])
			remoteAddr = fmt.Sprintf("%s:%s", tunnelSlice[2], tunnelSlice[3])
		}

		if localAddr == "" || remoteAddr == "" {
			log.Println("Warning: no local address or remote address found")
			continue
		}

		waitGroup.Add(1)
		go func() {
			err := p.RemoteProxy(protocol, localAddr, remoteAddr)
			if err != nil {
				log.Printf("remote proxy failed: %s - %v", tunnel, err)
			}
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
	return nil
}

func (p *Peer) LocalTunnels(tunnels ...string) error {

	var waitGroup sync.WaitGroup

	for _, tunnel := range tunnels {
		protocol := "tcp"
		localAddr := ""
		remoteAddr := ""
		tunnelDef := tunnel
		protoSlice := strings.Split(tunnel, "/")
		if len(protoSlice) == 2 {
			protocol = protoSlice[0]
			tunnelDef = protoSlice[1]
		}

		tunnelSlice := strings.Split(tunnelDef, ":")
		if len(tunnelSlice) == 3 {
			localAddr = fmt.Sprintf(":%s", tunnelSlice[0])
			remoteAddr = fmt.Sprintf("%s:%s", tunnelSlice[1], tunnelSlice[2])
		}
		if len(tunnelSlice) == 4 {
			localAddr = fmt.Sprintf("%s:%s", tunnelSlice[0], tunnelSlice[1])
			remoteAddr = fmt.Sprintf("%s:%s", tunnelSlice[2], tunnelSlice[3])
		}

		if localAddr == "" || remoteAddr == "" {
			log.Println("Warning: no local address or remote address found")
			continue
		}

		waitGroup.Add(1)
		go func() {
			err := p.LocalProxy(protocol, localAddr, remoteAddr)
			if err != nil {
				log.Printf("local proxy failed: %s - %v", tunnel, err)
			}
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
	return nil
}

func (p *Peer) RemoteProxy(proto, localAddress, remoteAddress string) error {
	if strings.HasPrefix(localAddress, ":") {
		ipSlice := strings.Split(p.LocalAddresses[0].String(), "/")
		localAddress = ipSlice[0] + localAddress
	}

	addrPort, err := netip.ParseAddrPort(localAddress)
	if err != nil {
		return err
	}

	log.Println("[p] listening on wireguard to ", addrPort)
	log.Println("[p] forwarding to ", remoteAddress)

	_, err = p.getDevice()
	if err != nil {
		return err
	}

	tcpAddr := net.TCPAddrFromAddrPort(addrPort)
	listener, err := p.tunnelNet.ListenTCP(tcpAddr)
	if err != nil {
		return err
	}

	defer listener.Close()
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}

		go (func() {
			serverConn, err := net.Dial(proto, remoteAddress)
			if err != nil {
				clientConn.Close()
				return
			}
			defer serverConn.Close()
			go (func() {
				_, err := io.Copy(serverConn, clientConn)
				if err != nil {
					log.Printf("Error copying from client to server: %v", err)
				}
			})()
			_, err = io.Copy(clientConn, serverConn)
			if err != nil {
				log.Printf("Error copying from server to client: %v", err)
			}
			err = clientConn.Close()
			if err != nil {
				log.Printf("Error closing client: %v", err)
			} else {
				log.Println("Closed client")
			}

		})()
	}
}

func (p *Peer) LocalProxy(proto, localAddress, remoteAddress string) error {

	listener, err := net.Listen(proto, localAddress)

	if err != nil {
		return err
	}

	defer listener.Close()
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go (func() {
			serverConn, err := p.Dial(proto, remoteAddress)
			if err != nil {
				log.Printf("Error copying from server to client: %v", err)
				clientConn.Close()
				return
			}
			defer serverConn.Close()
			go (func() {
				log.Printf("Error copying from server to client: %v", err)
				_, err := io.Copy(serverConn, clientConn)
				if err != nil {
					log.Printf("Error copying from client to server: %v", err)
				}
			})()
			_, err = io.Copy(clientConn, serverConn)
			if err != nil {
				log.Printf("Error copying from server to client: %v", err)
			}
			err = clientConn.Close()
			if err != nil {
				log.Printf("Error closing client: %v", err)
			} else {
				log.Println("Closed client")
			}

		})()
	}
}

func NewPeer(privateKey, serverPublicKey []byte, serverAddress string, localAddresses, dnsServers []netip.Addr, logLevel int) *Peer {
	return &Peer{
		PrivateKey:      privateKey,
		ServerPublicKey: serverPublicKey,
		Address:         serverAddress,
		Port:            DefaultPort,
		LogLevel:        logLevel,

		LocalAddresses: localAddresses,
		DNSServers:     dnsServers,
		MTU:            DefaultMTU,
	}
}
