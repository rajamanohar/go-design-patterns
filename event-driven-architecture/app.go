/*
**************************************************************************************
*
This project is licensed under the MIT license.
MIT License

# Copyright (c) 2023 Raja Manohar

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*
**************************************************************************************
*/
package main

import (
	"fmt"
	"net"
)

type Event struct {
	Type string
	Data interface{}
}

type EventHandler func(Event)

type EventBus struct {
	handlers map[string][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

func (eb *EventBus) Register(eventType string, handler EventHandler) {
	handlers := eb.handlers[eventType]
	handlers = append(handlers, handler)
	eb.handlers[eventType] = handlers
}

func (eb *EventBus) Dispatch(eventType string, data interface{}) {
	event := Event{Type: eventType, Data: data}
	handlers := eb.handlers[eventType]
	for _, handler := range handlers {
		handler(event)
	}
}

type ChatServer struct {
	eventBus *EventBus
	clients  map[net.Conn]bool
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		eventBus: NewEventBus(),
		clients:  make(map[net.Conn]bool),
	}
}

func (cs *ChatServer) Start(port string) error {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	fmt.Printf("Listening on port %s...\n", port)
	defer listener.Close()

	cs.eventBus.Register("new-connection", cs.onNewConnection)
	cs.eventBus.Register("disconnected", cs.onDisconnected)
	cs.eventBus.Register("message-received", cs.onMessageReceived)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		cs.eventBus.Dispatch("new-connection", conn)
	}
}

func (cs *ChatServer) onNewConnection(event Event) {
	conn := event.Data.(net.Conn)
	cs.clients[conn] = true
	fmt.Printf("New connection from %s\n", conn.RemoteAddr().String())
}

func (cs *ChatServer) onDisconnected(event Event) {
	conn := event.Data.(net.Conn)
	delete(cs.clients, conn)
	fmt.Printf("Disconnected from %s\n", conn.RemoteAddr().String())
}

func (cs *ChatServer) onMessageReceived(event Event) {
	msg := event.Data.(string)
	for conn := range cs.clients {
		_, err := conn.Write([]byte(msg))
		if err != nil {
			cs.eventBus.Dispatch("disconnected", conn)
		}
	}
}

type Client struct {
	conn     net.Conn
	eventBus *EventBus
}

func NewClient(conn net.Conn, eventBus *EventBus) *Client {
	return &Client{
		conn:     conn,
		eventBus: eventBus,
	}
}

func (c *Client) Start() {
	c.eventBus.Dispatch("new-connection", c.conn)

	buf := make([]byte, 1024)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			c.eventBus.Dispatch("disconnected", c.conn)
			break
		}

		msg := string(buf[:n])
		c.eventBus.Dispatch("message-received", msg)
	}
}

func main() {
	cs := NewChatServer()
	err := cs.Start(":8000")
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
