package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"tcp-app/logging"
)

const (
	host        = "localhost:"
	port        = "9999"
	kilobyte    = 1024
	socket_type = "tcp"
	logFilePath = "client_log.txt"
)

// struct that is for the quit channel,
type Quiter struct {
	Reason  string // is why the client connection is being closed
	Problem error  // can give the error so it will be logged
}

func readServerLoop(conn net.Conn, buffer []byte, resch chan string, quitch chan Quiter) {
sender:
	for {
		n, err := conn.Read(buffer)
		// any error just close the server connection
		if err != nil {
			quitch <- Quiter{
				Reason:  "ERROR",
				Problem: err,
			}
			break sender
		} else {
			serverRes := string(buffer[:n])
			resch <- serverRes
		}
	}
}

// reads data from given scanner and sends it thorugh the channel
func readUserInput(s *bufio.Scanner, out chan []byte, quitch chan Quiter) {
	// i need validate the user input
reader:
	for {
		fmt.Print("Message > ")
		s.Scan()
		data := s.Bytes()
		if len(data) <= 0 {
			fmt.Println("cannot send empty messages!")
			continue
		}
		if string(data) == "!DISCONNECT" {
			quitch <- Quiter{
				Reason:  "DISCONNECT SIGNAL",
				Problem: nil,
			}
			break reader
		} else {
			out <- data
		}
	}
}

func displayServerData(data string, log logging.Logger) {
	log.WriteToLogger(logging.INFO, "reciveing data of size:"+strconv.Itoa(len(data))+"bytes")
	fmt.Println("\nfrom server: ", data)
	// re promt the user
	fmt.Print("Message > ")
}
func handleClose(q Quiter, conn net.Conn, log logging.Logger) {
	username := conn.RemoteAddr().String()
	defer conn.Close()
	switch q.Reason {
	case "ERROR":
		log.WriteToLogger(logging.ERROR, "disconnected because of error: "+q.Problem.Error())
	case "DISCONNECT SIGNAL":
		// tell the server we disconnected
		conn.Write([]byte(fmt.Sprintf("user %s disconnected!", username)))
		log.WriteToLogger(logging.INFO, "user %s disconnected!"+username)
	default:
		log.WriteToLogger(logging.WARNING, "unknown reason for disconnect "+q.Problem.Error())
	}
}

func writeToTheServer(data []byte, conn net.Conn, log logging.Logger) {
	log.WriteToLogger(logging.INFO, "sending data of size:"+strconv.Itoa(len(data))+"bytes")
	conn.Write(data)
}

func handleChannels(conn net.Conn, sendch chan []byte, quitch chan Quiter, resch chan string, log logging.Logger) {
main:
	for {
		select {
		case d := <-sendch:
			writeToTheServer(d, conn, log)
			conn.Write(d)
		case r := <-resch:
			displayServerData(r, log)
		case q := <-quitch:
			// handle disconnect funciton
			handleClose(q, conn, log)
			break main
		}
	}
}
func initLoggers(cm logging.Logger) *logging.CustomLogger {
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("fatal error when creating log file" + err.Error())
	}
	InfoLogger := log.New(file, "INFO: ", log.Ldate|log.Ltime)
	WarningLogger := log.New(file, "WARNING: ", log.Ldate|log.Ltime)
	ErrorLogger := log.New(file, "ERROR: ", log.Ldate|log.Ltime)
	FatalLogger := log.New(file, "FATAL: ", log.Ldate|log.Ltime)
	cm.AddLogger(logging.INFO, InfoLogger)
	cm.AddLogger(logging.WARNING, WarningLogger)
	cm.AddLogger(logging.ERROR, ErrorLogger)
	cm.AddLogger(logging.FATAL, FatalLogger)
	return cm.(*logging.CustomLogger)
}

func main() {
	cm := logging.NewLogger()
	log := initLoggers(cm)
	conn, err := net.Dial(socket_type, host+port)
	if err != nil {
		log.WriteToLogger(logging.FATAL, "there was an error when connecting to:"+err.Error())
	}
	log.WriteToLogger(logging.INFO, "connected to the server")
	tosendch := make(chan []byte)
	resch := make(chan string)
	quitch := make(chan Quiter)
	buff := make([]byte, 1024)
	scanner := bufio.NewScanner(os.Stdin)
	go readServerLoop(conn, buff, resch, quitch)
	go readUserInput(scanner, tosendch, quitch)
	// run this loop on main channel to handle all of the channels
	handleChannels(conn, tosendch, quitch, resch, log)
}
