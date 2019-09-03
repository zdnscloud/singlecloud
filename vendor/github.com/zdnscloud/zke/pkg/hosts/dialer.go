package hosts

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/zdnscloud/zke/pkg/k8s"

	"golang.org/x/crypto/ssh"
)

const (
	DockerDialerTimeout = 120
)

type DialerFactory func(h *Host) (func(network, address string) (net.Conn, error), error)

type dialer struct {
	signer       ssh.Signer
	sshKeyString string
	sshAddress   string
	username     string
	netConn      string
	dockerSocket string
}

type DialersOptions struct {
	DockerDialerFactory DialerFactory
	K8sWrapTransport    k8s.WrapTransport
}

func GetDialerOptions(d DialerFactory, w k8s.WrapTransport) DialersOptions {
	return DialersOptions{
		DockerDialerFactory: d,
		K8sWrapTransport:    w,
	}
}

func NewDialer(h *Host, kind string) (*dialer, error) {
	dialer := &dialer{
		sshAddress:   fmt.Sprintf("%s:%s", h.Address, h.Port),
		username:     h.User,
		dockerSocket: h.DockerSocket,
		sshKeyString: h.SSHKey,
		netConn:      "unix",
	}

	if dialer.sshKeyString == "" {
		var err error
		dialer.sshKeyString, err = PrivateKeyPath(h.SSHKeyPath)
		if err != nil {
			return nil, err
		}
	}

	switch kind {
	case "network", "health":
		dialer.netConn = "tcp"
	}

	if len(dialer.dockerSocket) == 0 {
		dialer.dockerSocket = "/var/run/docker.sock"
	}

	return dialer, nil
}

func SSHFactory(h *Host) (func(network, address string) (net.Conn, error), error) {
	dialer, err := NewDialer(h, "docker")
	return dialer.Dial, err
}

func LocalConnFactory(h *Host) (func(network, address string) (net.Conn, error), error) {
	dialer, err := NewDialer(h, "network")
	return dialer.Dial, err
}

func (d *dialer) Dial(network, addr string) (net.Conn, error) {
	var conn *ssh.Client
	var err error
	conn, err = d.getSSHTunnelConnection()
	if err != nil {
		if strings.Contains(err.Error(), "no key found") {
			return nil, fmt.Errorf("Unable to access node with address [%s] using SSH. Please check if the configured key or specified key file is a valid SSH Private Key. Error: %v", d.sshAddress, err)
		} else if strings.Contains(err.Error(), "no supported methods remain") {
			return nil, fmt.Errorf("Unable to access node with address [%s] using SSH. Please check if you are able to SSH to the node using the specified SSH Private Key and if you have configured the correct SSH username. Error: %v", d.sshAddress, err)
		} else if strings.Contains(err.Error(), "cannot decode encrypted private keys") {
			return nil, fmt.Errorf("Unable to access node with address [%s] using SSH. Using encrypted private keys is only supported using ssh-agent. Please configure the option `ssh_agent_auth: true` in the configuration file or use --ssh-agent-auth as a parameter when running ZKE. This will use the `SSH_AUTH_SOCK` environment variable. Error: %v", d.sshAddress, err)
		} else if strings.Contains(err.Error(), "operation timed out") {
			return nil, fmt.Errorf("Unable to access node with address [%s] using SSH. Please check if the node is up and is accepting SSH connections or check network policies and firewall rules. Error: %v", d.sshAddress, err)
		}
		return nil, fmt.Errorf("Failed to dial ssh using address [%s]: %v", d.sshAddress, err)
	}

	// Docker Socket....
	if d.netConn == "unix" {
		addr = d.dockerSocket
		network = d.netConn
	}

	remote, err := conn.Dial(network, addr)
	if err != nil {
		if strings.Contains(err.Error(), "connect failed") {
			return nil, fmt.Errorf("Unable to access the service on %s. The service might be still starting up. Error: %v", addr, err)
		} else if strings.Contains(err.Error(), "administratively prohibited") {
			return nil, fmt.Errorf("Unable to access the Docker socket (%s). Please check if the configured user can execute `docker ps` on the node, and if the SSH server version is at least version 6.7 or higher. If you are using RedHat/CentOS, you can't use the user `root`. Please refer to the documentation for more instructions. Error: %v", addr, err)
		}
		return nil, fmt.Errorf("Failed to dial to %s: %v", addr, err)
	}
	return remote, err
}

func (d *dialer) getSSHTunnelConnection() (*ssh.Client, error) {
	cfg, err := GetSSHConfig(d.username, d.sshKeyString)
	if err != nil {
		return nil, fmt.Errorf("Error configuring SSH: %v", err)
	}
	// Establish connection with SSH server
	return ssh.Dial("tcp", d.sshAddress, cfg)
}

func (h *Host) newHTTPClient(dialerFactory DialerFactory) (*http.Client, error) {
	factory := dialerFactory
	if factory == nil {
		factory = SSHFactory
	}

	dialer, err := factory(h)
	if err != nil {
		return nil, err
	}
	dockerDialerTimeout := time.Second * DockerDialerTimeout
	return &http.Client{
		Transport: &http.Transport{
			Dial:                  dialer,
			TLSHandshakeTimeout:   dockerDialerTimeout,
			IdleConnTimeout:       dockerDialerTimeout,
			ResponseHeaderTimeout: dockerDialerTimeout,
		},
	}, nil
}
