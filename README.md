# Wireguard Application Tunneller

Create tunnels with SSH like syntax using a binaries that have wireguard keys baked in them. Simply drop to disk and run.

## Compiling

```bash
# set these variables to your values
PrivateKeyB64=""
ServerPublicKeyB64=""
ServerAddress=""
TunnelLocalAddress=""
TunnelDNSServer=""

LocalTunnels="8080:10.4.0.1:8888,2222:10.4.0.1:22"
RemoteTunnels="443:google.com:443"

pkg="github.com/defektive/wat/pkg/cmd"

```

Build

```bash
CGO_ENABLED=0; go build -ldflags "-X $pkg.defaultLocalTunnels=$LocalTunnels -X $pkg.defaultRemoteTunnels=$RemoteTunnels -X $pkg.defaultPrivateKeyB64=$PrivateKeyB64 -X $pkg.defaultServerPublicKeyB64=$ServerPublicKeyB64 -X $pkg.defaultServerAddress=$ServerAddress -X $pkg.defaultTunnelLocalAddress=$TunnelLocalAddress -X $pkg.defaultTunnelDNSServer=$TunnelDNSServer"
```



## Usage

Expose port 3306 on localhost and tunnel all traffic to port 3306 on a host on the wireguard network.
```bash
./wat proxy -L 127.0.0.1:3306:10.4.0.1:3306
```

Expose port 2222 on broadcast and tunnel all traffic to port 22 on a host on the wireguard network.
```bash
./wat proxy -L 2222:10.4.0.1:22
```

Accept traffic from wireguard network on port 443, forward it to port 443 on the local network.
```bash
./wat proxy -L 443:intranet.example.com:443
```

Chain multiple tunnels

```bash
./wat proxy -L 127.0.0.1:8080:10.4.0.1:8888 -L 3333:10.4.0.1:22 -R 8080:10.28.0.5:8080
```

## To Do Items

- [ ] Allow specifying keys and endpoint from the CLI
- [ ] github releases
- [ ] Accept tunnels from STDIN
- [ ] Add command aliases  `tunnel`, `p`, `run`
- [ ] tests
- [ ] improved logging (slog)
- [ ] Allow changing tunnels while running
- [ ] Impress my friends by getting more stars
- [ ] create an `xwat` builder CLI that will generate new configs, add them to the wireguard server, then build wat with the appropriate values.
- [ ] Using logic from `xwat` create an http(s) service.
  - [ ] authenticate requests with tokens
  - [ ] compile and deliver binaries 
  - [ ] generate configs to pass to `wat` if it was started without keys.
- [ ] Socks proxy
- [ ] userspace wireguard server


## Fun helpers

Install wireguard on Ubuntu
```bash
sudo apt install wireguard
```

Generate keys
```bash
wg genkey | sudo tee private.key | wg pubkey | sudo tee public.key
```

Setup wireguard server
```bash

if ! sudo test -f /etc/wireguard/private.key ; then
  echo "Generating new pub/private keys"
  wg genkey | sudo tee /etc/wireguard/private.key | wg pubkey | sudo tee /etc/wireguard/public.key
else
  echo "Skipping key generation since file exists"
fi

cat << EOF | sudo tee -a /etc/wireguard/wg0.conf
[Interface]
Address = 10.250.0.1/24
SaveConfig = true
ListenPort = 51820
PrivateKey = $(sudo cat /etc/wireguard/private.key)
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -t nat -A POSTROUTING -o ens5 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -t nat -D POSTROUTING -o ens5 -j MASQUERADE

EOF

if ! grep -P "^net\.ipv4\.ip_forward=1$" /etc/sysctl.conf ; then 
  echo "Configuring IP forwarding"
  echo | sudo tee -a /etc/sysctl.conf
  echo "# forward packets for wireguard" | sudo tee -a /etc/sysctl.conf
  echo "net.ipv4.ip_forward=1" | sudo tee -a /etc/sysctl.conf
  
  sudo sysctl -p
fi

sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0
```

Setup wireguard ubuntu client.
Run this on the server, then copy and past the result to the client


```bash

NEXT_IP_NUM=$(( $(sudo wg show | grep "peer" | wc -l) + 10 ))
SERVER_PRIVATE_IP=$(ip -4 a show ens5 | grep inet | awk '{print $2}')
NET=$(sudo cat /etc/wireguard/wg0.conf | grep Address | awk '{print$NF}' | sed 's|1/24|0/24|')
NEXT_IP=$(echo $NET | sed 's|0/24|'$NEXT_IP_NUM'/32|')


echo "## Begin copy and pastable script ##"

cat << EOF

if ! sudo test -f /etc/wireguard/private.key ; then
  echo "Generating new pub/private keys"
  wg genkey | sudo tee /etc/wireguard/private.key | wg pubkey | sudo tee /etc/wireguard/public.key
else
  echo "Skipping key generation since file exists"
fi

cat << EOFC | sudo tee -a /etc/wireguard/wg0.conf
[Interface]
PrivateKey = \$(sudo cat /etc/wireguard/private.key)
Address = $NEXT_IP

[Peer]
PublicKey =  $(sudo cat /etc/wireguard/public.key)
AllowedIPs = $NET
Endpoint = $(curl -s ipcurl.net):$(sudo cat /etc/wireguard/wg0.conf | grep ListenPort | awk '{print$NF}')

EOFC


echo "no you can add the peer to the server"
echo "sudo wg set wg0 peer \$(sudo cat /etc/wireguard/public.key)  allowed-ips $NEXT_IP"
echo or
echo "ssh -o StrictHostKeyChecking=no $(echo $SERVER_PRIVATE_IP | cut -d/ -f1) \"sudo wg set wg0 peer \$(sudo cat /etc/wireguard/public.key)  allowed-ips $NEXT_IP\""


sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0

EOF

echo "## End copy and pastable script ##"


```

