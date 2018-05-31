package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	login    = "pavel3"
	pass     = "lol"
	dataPath = "./data.txt"
)

type MsgQuery struct {
	Receiver string `json:"rec"`
	Message  string `json:"msg"`
}

type calcQuery struct {
	Function string `json:"func"`
	Data     []byte `json:"data"`
}

func Login(name, pass string, conn net.Conn, reader bufio.Reader) {
	conn.Write([]byte("I{\"login\":\"" + name + "\",\"pass\":\"" + pass + "\"}\n"))
	select {
	case ok := <-okChan:
		fmt.Println(ok)
	case err := <-errChan:
		panic(err)
	}
}

func SendMessage(conn net.Conn, reader bufio.Reader) {
	fmt.Print("\nEnter receiver: ")
	res, _ := reader.ReadString('\n')
	fmt.Print("Enter message: ")
	msg, _ := reader.ReadString('\n')
	m := &MsgQuery{
		Receiver: strings.TrimSpace(res),
		Message:  strings.TrimSpace(msg),
	}
	byt, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	conn.Write([]byte("M"))
	conn.Write(byt)
	conn.Write([]byte("\n")) // always add \n at the end!s

	select {
	case ok := <-okChan:
		fmt.Println(ok)
	case err := <-errChan:
		fmt.Println(err)
	}
}

func StreamMessage(login string, conn net.Conn, reader bufio.Reader) {
	fmt.Print("\nEnter message: ")
	msg, _ := reader.ReadString('\n')
	m := &MsgQuery{
		Receiver: login, // sender here
		Message:  strings.TrimSpace(msg),
	}
	byt, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	conn.Write([]byte("S"))
	conn.Write(byt)
	conn.Write([]byte("\n")) // always add \n at the end!

	select {
	case ok := <-okChan:
		fmt.Println(ok)
	case err := <-errChan:
		fmt.Println(err)
	}
}

func ListMessages(reader bufio.Reader) {
	empty := true
LOOP:
	for {
		select {
		case m := <-msgChan:
			fmt.Println("From: ", m.Receiver, "\nContent: ", m.Message)
			empty = false
		default:
			if empty {
				fmt.Println("No messages")
			}
			break LOOP
		}
	}
}

func DeclareAndReady(conn net.Conn, console bufio.Reader) {
	fmt.Print("\nEnter func name: ")
	f, _ := console.ReadString('\n')

	conn.Write([]byte("P{\"func\":\"" + f[:len(f)-1] + "\"}\n"))

	select {
	case ok := <-okChan:
		fmt.Println(ok)
	case err := <-errChan:
		fmt.Println(err)
		return
	}

	conn.Write([]byte("R\n"))
	for {
		// q to exit
		data := <-calcChan
		file, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(err)
		}
		_, err = file.Write(data)
		if err != nil {
			panic(err)
		}
		// execute
		// check for done file
		// send answer to conn

		//fmt.Println(string(data))
	}
}

func CalculateFunc(conn net.Conn, console bufio.Reader) {
	fmt.Print("\nEnter func name: ")
	f, _ := console.ReadString('\n')

	// hardcode here, only for example
	fmt.Print("\nEnter vector components: ")
	comp, _ := console.ReadString('\n')
	fmt.Print("\nEnter scalar: ")
	sc, _ := console.ReadString('\n')

	c := &calcQuery{
		Function: f[:len(f)-1],
		Data:     []byte("{\"vector\":[" + comp[:len(comp)-1] + "],\"scalar\":" + sc[:len(sc)-1] + "}"),
	}
	// end of hardcode

	byt, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	conn.Write([]byte("C"))
	conn.Write(byt)
	conn.Write([]byte("\n")) // always add \n at the end!

	select {
	case ok := <-doneChan:
		fmt.Println(string(ok)) // TODO
	case err := <-errChan:
		fmt.Println(err)
	}
}

func ReadMessages(reader bufio.Reader) {
	for {
		msgType, err := reader.ReadByte()
		msg, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Second)
				panic("server has closed your conn")
			}
			panic(err)
		}
		if msgType == 'E' {
			errChan <- string(msg)
		} else if msgType == 'M' {
			m := &MsgQuery{}
			err = json.Unmarshal(msg, m)
			msgChan <- m
		} else if msgType == 'O' {
			okChan <- string(msg)
		} else if msgType == 'D' {
			doneChan <- msg
		} else if msgType == 'C' {
			calcChan <- msg
		} else {
			fmt.Println("\n[IN] Unexpected message type!", string(msgType), string(msg))
		}
	}
}

var (
	okChan   = make(chan string, 10)
	errChan  = make(chan string, 10)
	msgChan  = make(chan *MsgQuery, 10)
	calcChan = make(chan []byte, 10)
	doneChan = make(chan []byte, 10)
)

func main() {
	conn, err := net.Dial("tcp", ":8080") //"195.19.32.74:2018")
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(conn)
	console := bufio.NewReader(os.Stdin)
	go ReadMessages(*reader)

	// login first if you have already registered
	Login(login, pass, conn, *reader)

	for {
		fmt.Print("\n------------\nClient menu:\n1) Send message\n2) Check messages\n3) Stream message\n4) Declare func and execute\n5) Calculate func\n------------\n\nEnter number: ")

		keyStr, _ := console.ReadString('\n')
		key, err := strconv.Atoi(keyStr[:len(keyStr)-1])
		if err != nil {
			continue
		}

		switch key {
		case 1:
			SendMessage(conn, *console)
		case 2:
			ListMessages(*reader)
		case 3:
			StreamMessage(login, conn, *console)
		case 4:
			// bool declared
			DeclareAndReady(conn, *console)
		case 5:
			CalculateFunc(conn, *console)
		default:
			fmt.Println("No such option!")
		}
	}
	conn.Close()
}
