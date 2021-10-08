package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func typeCommand() {
	time.Sleep(100 * time.Millisecond)
	fmt.Print("Type command: ")
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal("Specify server port in arguments!")
	}
	port := ":" + os.Args[1]
	conn, err := net.Dial("tcp", port)
	if err != nil {
		log.Fatal("Server is not available check connection!")
	}
	server := NewServer(conn)

	reader := NewReader(bufio.NewReader(os.Stdin))

	fmt.Print("Type nickname: ")
	nick, err := reader.HandleNick()
	if err != nil {
		fmt.Println(err)
		return
	}

	server.Write("REG %s\n", string(nick))

	for {
		go server.Listen()
		go typeCommand()

		msg, err := reader.getMsgCommand()
		if err != nil {
			if !errors.Is(err, accident) {
				fmt.Println(err)
			}
			continue
		}

		if strings.TrimSpace(msg.Str) == "STOP" {
			fmt.Println("Exiting TCP server!")
			return
		}
		server.Write(msg.Str)
	}
}

// Server
type Server struct {
	Conn net.Conn
	c    chan bool
}

func NewServer(c net.Conn) *Server {
	return &Server{
		Conn: c,
		c:    make(chan bool, 1),
	}
}

func (s *Server) Listen() {
	for {
		msg, err := bufio.NewReader(s.Conn).ReadBytes('\n')
		msg, err = decode(msg)
		switch {
		case err == io.EOF:
			log.Fatal("Disconnected from server")
		case err != nil:
			fmt.Println("\nError while reading from server", err)
		default:
			fmt.Printf("\nMessage from server: %s", msg)
		}
	}
}

func (s *Server) Write(format string, a ...interface{}) {
	if _, err := fmt.Fprintf(s.Conn, format, a...); err != nil {
		fmt.Println("Could not write to server")
	}
}

type Message struct {
	Len     int
	Content []byte
	Str     string
}

var accident = errors.New("accident")

type Reader struct {
	obj *bufio.Reader
}

func NewReader(r *bufio.Reader) *Reader {
	return &Reader{obj: r}
}

func (r *Reader) HandleNick() ([]byte, error) {
	nick, _ := r.FromStdIn()
	nick = bytes.TrimSpace(nick)
	if len(nick) < 1 {
		return nil, errors.New("Please provide valid nickname!")
	}
	return nick, nil
}

// For simplicity only one command is available
func (r *Reader) getMsgCommand() (Message, error) {
	text, err := r.FromStdIn()
	if err != nil {
		return Message{}, err
	}
	args := bytes.TrimSpace(text)
	if len(args) == 0 {
		return Message{}, accident
	}

	split := bytes.Split(args, []byte(" "))

	cmd := bytes.TrimSpace(split[0])
	if string(cmd) != "MSG" {
		return Message{}, errors.New("Command starts with MSG")
	}

	args = bytes.TrimSpace(bytes.TrimPrefix(args, cmd))
	if len(args) < 1 {
		return Message{}, errors.New("Specify recipient")
	}

	recipient := bytes.TrimSpace(split[1])
	msg := bytes.TrimSpace(bytes.TrimPrefix(args, recipient))
	if len(msg) < 1 {
		return Message{}, errors.New("Specify message for recipient")
	}
	str := fmt.Sprintf("%s %s %d//%s\n", cmd, recipient, len(string(msg)), msg)

	return Message{
		Str:     string(str),
		Content: text,
		Len:     len(msg),
	}, nil
}

func (r *Reader) FromStdIn() ([]byte, error) {
	text, err := r.obj.ReadBytes('\n')
	if err != nil {
		return nil, errors.New("Text is unredable, retype!")
	}
	return text[:len(text)-1], nil
}

const DELIMETER = "//"

var Splitter = []byte(DELIMETER)

func decode(b []byte) ([]byte, error) {
	bodyLen := bytes.Split(b, Splitter)[0]
	length, err := strconv.Atoi(string(bodyLen))
	if err != nil {
		return nil, fmt.Errorf("no message body")
	}
	if length == 0 {
		return nil, fmt.Errorf("message body is empty")
	}

	padding := len(bodyLen) + len(Splitter)
	body := append(b[padding:padding+length], []byte("\n")...)
	return body, nil
}
