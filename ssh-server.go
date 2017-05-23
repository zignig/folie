package folie

// This file contains the SSH server.

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

// SSHServer represents an instance of an SSH server that accepts incoming connections that
// gain access to the serial port managed by folie.
type SSHServer struct {
	listener    net.Listener
	sshConfig   *ssh.ServerConfig
	addTxWriter func(io.Writer)
}

// NewSSHServer creates a new SSHServer, opens the listening socket, and validates that the
// server key and authorized keys files can be read.
func NewSSHServer(listenAddr, serverKeyFile, authorizedKeysFile string) (*SSHServer, error) {
	config := &ssh.ServerConfig{}

	// Set-up authorized client keys.
	if authorizedKeysFile == "insecure" {
		config.NoClientAuth = true
	} else {
		keyMap, err := readAuthorizedKeys(authorizedKeysFile)
		if err != nil {
			return nil, err
		}
		config.PublicKeyCallback = func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if _, ok := keyMap[string(pubKey.Marshal())]; ok {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		}
	}

	// Set-up host key.
	privateBytes, err := ioutil.ReadFile(serverKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load host key from %s: %s", serverKeyFile, err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host key from %s: %s", serverKeyFile, err)
	}
	config.AddHostKey(private)

	// Create the listener socket.
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %s", listenAddr, err)
	}

	return &SSHServer{listener: listener, sshConfig: config}, nil
}

// Run is an infinite loop that accepts incoming connections. For each connection it starts a
// goroutine that reads on the connection and pushes bytes into the rx channel (which is shared
// across all). It also makes a callback to ss.addTxWriter to register the SSH channel with the
// switchboard for transmission.
func (ss *SSHServer) Run(rx chan<- []byte, addTxWriter func(io.Writer)) {
	ss.addTxWriter = addTxWriter
	// Run the accept loop, it ends with os.Exit...
	for {
		// Accept a connection.
		conn, err := ss.listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal SSH listener error: %s", err)
			continue
		}
		fmt.Fprintf(os.Stderr, "\n[Accepted SSH from %s]\n", conn.RemoteAddr())

		// Start goroutine to service the connection.
		go ss.service(conn, rx)
	}
}

//

// readAuthorizedKeys reads an authorized keys file and returns a hash with the keys.
func readAuthorizedKeys(file string) (map[string]struct{}, error) {
	authorizedKeysBytes, err := ioutil.ReadFile("authorized_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to load authorized keys from %s: %v", file, err)
	}

	authorizedKeysMap := map[string]struct{}{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing authorized keys from %s: %v", file, err)
		}

		authorizedKeysMap[string(pubKey.Marshal())] = struct{}{}
		authorizedKeysBytes = rest
	}

	return authorizedKeysMap, nil
}

// service initalizes a connection and then services it.
func (ss *SSHServer) service(conn net.Conn, rx chan<- []byte) { //, cmd chan string) {
	// Perform SSH handshake. newChan is a channel where new SSH channel open requests come int
	// and reqChan is where out-of-band requests come in.
	_, newChan, reqChan, err := ssh.NewServerConn(conn, ss.sshConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed SSH handshake: %s\n", err)
		return // the connection is already closed by NewServerConn
	}

	// We discard incoming requests at the connection level.
	go ssh.DiscardRequests(reqChan)

	// Service the incoming newChan channel.
	for newChannel := range newChan {
		// Channels have a type, depending on the application level protocol intended.
		// In the case of a shell, the type is "session".
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error accepting SSH channel: %s\n", err)
			continue
		}

		// Incoming requests are used for out-of-band commands, for example to reset the
		// attached uC or change the baud rate. We also need to handle the "shell" request
		// so one can connect to folie using a std SSH client.
		go func() {
			for req := range requests {
				switch req.Type {
				case "shell": // used by std SSH clients to get started
					req.Reply(true, nil)
				case "env": // used by std SSH client
					req.Reply(true, nil)
				default:
					fmt.Fprintf(os.Stderr, "unknown SSH request: %s\n", req.Type)
					req.Reply(false, nil)
				}
			}
		}()

		// Service incoming SSH data and forward to the serial port.
		go func() {
			defer channel.Close()
			for {
				// Read data from SSH channel
				buf := getBuffer()
				n, err := channel.Read(buf)
				if n > 0 {
					rx <- buf[:n]
					continue
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "error reading from SSH channel: %s\n", err)
					return
				}
			}
		}()

		// Register with switchboard so it can TX data.
		ss.addTxWriter(channel)
	}
}
