package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"tcp-app/logging"
)

const (
	serverType    string = "tcp"
	listenAddress string = "localhost:9999"
	bufferSize    uint   = 1024
	channelBuffer uint   = 10
	logFilePath          = "server_log.txt"
)

// structure of the data recived from the client
type Message struct {
	from    string
	payload []byte
}
type Server struct {
	listenAddress string
	ln            net.Listener
	quitch        chan struct{}
	msgch         chan Message
	connections   ServerMapper
	log           logging.Logger
}

func NewServer(listenAdder string, mapper ServerMapper, log logging.Logger) *Server {
	return &Server{
		listenAddress: listenAdder,
		quitch:        make(chan struct{}),
		//buffer the channel so its not blocking
		msgch:       make(chan Message, channelBuffer),
		connections: mapper,
		log:         log,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen(serverType, s.listenAddress)
	if err != nil {
		s.log.WriteToLogger(logging.ERROR, "server failed to start", err)
		return err
	}
	s.log.WriteToLogger(logging.INFO, "server has successfully started")
	s.ln = ln
	go s.acceptLoop()
	//since channels are blocking none of the code below will be touched
	//until something is recive through the channel
	//or the channel is closed
	<-s.quitch
	err = s.killServer()
	if err != nil {
		s.log.WriteToLogger(logging.FATAL, "unknow error when killing server", err)
	}
	s.log.GetLogger(logging.INFO).Println("the server was successfully closed")
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accecpt error", err)
			// ignore this error because it will be handled in the readLoop
			continue
		}
		s.log.WriteToLogger(logging.INFO, "user connected:"+conn.RemoteAddr().String())
		go s.readLoop(conn)
	}
}
func (s *Server) readLoop(conn net.Conn) {
	buff := make([]byte, bufferSize)
	for {
		n, err := conn.Read(buff)
		if err != nil {
			s.handleDisconnect(conn, err)
		}
		// return if no data was sent
		if n == 0 {
			s.log.WriteToLogger(logging.WARNING, "empty data recieved")
			return
		}
		msg := Message{
			from:    conn.RemoteAddr().String(),
			payload: buff[:n],
		}
		strmsg := string(msg.payload)
		//if the message is equal to kill then write to the quit chan which
		//will call the kill server func
		if strmsg == "!KILL" {
			close(s.quitch)
		}
		s.connections.Add(msg.from, conn)
		s.msgch <- msg
	}
}
func (s *Server) broadcast() {
	for msg := range s.msgch {
		fmt.Printf("reviced message from connection from: %s\n > %s\n", msg.from, string(msg.payload))
		for from, conn := range s.connections.AllConns() {
			conn.Write([]byte(from))
			conn.Write(msg.payload)
		}
	}
}

func (s *Server) killServer() error {
	//make sure to close all of the client connections so they arent left hanging
	for _, conn := range s.connections.AllConns() {
		conn.Close()
	}
	close(s.msgch)
	err := s.ln.Close()
	if err != nil {
		s.log.WriteToLogger(logging.ERROR, "problem when killing server", err)
		return err
	}
	return nil

}

// handle the disconnect of the client from the server
// optional error can be given
func (s *Server) handleDisconnect(conn net.Conn, userErr ...error) {
	fmt.Println("disconnect from user")
	user := conn.RemoteAddr().String()
	s.connections.Remove(user)
	conn.Close()
	// also will need to add a case for when disconnected when message limit reached
	if len(userErr) == 1 {
		// this checks for a specific reason for the disconnect
		switch errors.Unwrap(userErr[0]) {
		case net.ErrClosed:
			// ignore these errors, becueace one send of empty data will be sent when conn is disconnected
			return
		default:
			// this should be the catch for the EOF error
			s.log.WriteToLogger(logging.INFO, "user disconnected successfully")
		}
	} else {
		s.log.WriteToLogger(logging.WARNING, "user disconnected for unknow reason", userErr...)
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
	customLogger := logging.NewLogger()
	logger := initLoggers(customLogger)
	safeMap := NewSafeMap()
	server := NewServer(listenAddress, &safeMap, logger)
	go server.broadcast()
	server.Start()
}
