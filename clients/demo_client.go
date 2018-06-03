package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	login = os.Args[1]
	pass  = os.Args[2]
)

const (
	address = "195.19.32.74:2018"

	binaryFuncPath = "./func"          // your math function
	dataReadyPath  = "./dataReady.txt" // file where income args for your func execution will be stored

	binaryCalcPath = "./calc"         // program which asks params for your func and stores them in a single json
	dataCalcPath   = "./dataCalc.txt" // file where calc will store json-argument (for filed "data" in C request)

	binaryResultPath = "./result" // prigram which outputs result (json-answer is passed as first parameter)
)

type MsgQuery struct {
	Receiver string `json:"rec"`
	Message  string `json:"msg"`
}

type calcQuery struct {
	Function string `json:"func"`
	Data     []byte `json:"data"`
}

func Login(name, pass string, conn net.Conn) {
	conn.Write([]byte("I{\"login\":\"" + name + "\",\"pass\":\"" + pass + "\"}\n"))
	select {
	case ok := <-okChan:
		fmt.Println(ok)
	case err := <-errChan:
		panic(err)
	}
}

func SendMessage(conn net.Conn) {
	fmt.Print("\nEnter receiver: ")
	var res, msg string
	_, err := fmt.Scan(&res)
	fmt.Print("Enter message: ")
	_, err = fmt.Scan(&msg)
	if err != nil {
		panic(err)
	}
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

func StreamMessage(login string, conn net.Conn) {
	fmt.Print("\nEnter message: ")
	var msg string
	_, err := fmt.Scan(&msg)
	if err != nil {
		panic(err)
	}
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

func ListMessages() {
	empty := true
LOOP:
	for {
		select {
		case m := <-msgChan:
			fmt.Println("+\n|From: ", m.Receiver, "\n|Content:\n|\t", m.Message, "\n+")
			empty = false
		default:
			if empty {
				fmt.Println("No messages")
			}
			break LOOP
		}
	}
}

func Declare(conn net.Conn, f string) bool {
	conn.Write([]byte("P{\"func\":\"" + f + "\"}\n"))

	select {
	case ok := <-okChan:
		fmt.Println(ok)
		return true
	case err := <-errChan:
		fmt.Println(err)
		return false
	}
}

func Ready(conn net.Conn) {
	conn.Write([]byte("R\n"))
	for {
		data := <-calcChan
		file, err := os.OpenFile(dataReadyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(err)
		}
		_, err = file.Write(data)
		if err != nil {
			panic(err)
		}

		out, err := exec.Command(binaryFuncPath, dataReadyPath).Output()
		if err != nil {
			panic(err)
		}

		conn.Write([]byte("D"))
		conn.Write(out)
		conn.Write([]byte("\n")) // always add \n at the end!
	}
}

func CalculateFunc(conn net.Conn) {
	fmt.Print("\nEnter func name: ")
	var f string
	_, err := fmt.Scan(&f)
	fmt.Println("enter:", f)

	file, err := os.Create(dataCalcPath)
	if err != nil {
		panic(err)
	}
	file.Close()
	cmd := exec.Command(binaryCalcPath, dataCalcPath)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	file, err = os.OpenFile(dataCalcPath, os.O_RDONLY, 0666)
	out, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	file.Close()

	c := &calcQuery{
		Function: f,
		Data:     out,
	}

	byt, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	conn.Write([]byte("C"))
	conn.Write(byt)
	conn.Write([]byte("\n")) // always add \n at the end!

	select {
	case ok := <-doneChan:
		cmd := exec.Command(binaryResultPath, string(ok))
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
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
			time.Sleep(time.Second)
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
	okChan   = make(chan string, 1)
	errChan  = make(chan string, 1)
	msgChan  = make(chan *MsgQuery, 10)
	calcChan = make(chan []byte, 1)
	doneChan = make(chan []byte, 1)
)

func main() {
	conn, err := net.Dial("tcp", address)
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(conn)
	go ReadMessages(*reader)

	// login first if you have already registered
	Login(login, pass, conn)

	declared := false
	ready := false

	for {
		fmt.Printf("\n------------\nYou've logged as %s\nClient menu:\n1) Send message\n2) Check messages\n3) Stream message\n4) Declare and exec my func\n5) Calculate someone's func\n6) Exit\n------------\n\nEnter number: ", login)
		key := 0
		_, err := fmt.Scan(&key)
		if err != nil {
			continue
		}

		switch key {
		case 1:
			SendMessage(conn)
		case 2:
			ListMessages()
		case 3:
			StreamMessage(login, conn)
		case 4:
			if !declared {
				fmt.Print("\nEnter func name: ")
				f := "unnamed"
				_, err := fmt.Scan(&f)
				if err != nil {
					panic(err)
				}
				fmt.Println(f)
				declared = Declare(conn, f)
			}
			if !ready {
				ready = true
				go Ready(conn)
			}
		case 5:
			CalculateFunc(conn)
		case 6:
			fmt.Println("Bye!")
			return
		default:
			fmt.Println("No such option!")
		}
	}
}
