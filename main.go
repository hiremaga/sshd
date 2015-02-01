package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	log.SetOutput(os.Stderr)

	config := &ssh.ServerConfig{
		PasswordCallback: authenticate,
	}

	privateKey, err := readHostKey()
	if err != nil {
		log.Fatalln("Failed to read private key: ", err)
	}
	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT")))
	if err != nil {
		log.Fatalln("Failed to listen: ", err)
	}

	conn, err := listener.Accept()
	if err != nil {
		log.Fatalln("Failed to accept incoming connection: ", err)
	}

	_, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Fatalln("Failed to handshake: ", err)
	}

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			log.Println("Rejected unknown channel type: ", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")

			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			//TODO: Should this panic since it's a per-connection failure?
			log.Fatalln("Could not accept channel: ", err)
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false

				switch req.Type {
				case "shell":
					log.Println("Received shell request")
					ok = true
					if len(req.Payload) > 0 {
						ok = false
					}
				case "pty-req":
					log.Println("Received pty-req request")
					ok = true
				case "env":
					log.Println("Received env request")
					ok = true
				default:
					log.Println("Did not understand request type: ", req.Type)
				}

				//TODO: What is the second nil argument about?
				req.Reply(ok, nil)
			}
		}(requests)

		term := terminal.NewTerminal(channel, "> ")

		go func() {
			defer channel.Close()
			for {
				line, err := term.ReadLine()
				if err != nil {
					log.Println("Breaking from readline loop: ", err)
					break
				}
				fmt.Println(line)
			}
		}()
	}
}

func authenticate(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	//TODO: Authenticate provided credentials
	return nil, nil
}

func readHostKey() (ssh.Signer, error) {
	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		return nil, err
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, err
	}

	return private, nil
}
