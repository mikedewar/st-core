package core

import (
	"errors"
	"log"
	"net"
)

func UDPClientInterface() SourceSpec {
	return SourceSpec{
		Name: "UDPClient",
		Type: UDPCLIENT,
		New:  NewUDPClient,
	}
}

type UDPClientconf struct {
	c    chan error
	addr string
}

type UDPClientSend struct {
	msg string
	c   chan error
}

type UDPClient struct {
	conn          *net.UDPConn
	dial          chan UDPClientconf
	send          chan chan error
	recv          chan chan error
	quit          chan chan error
	fromUDPClient chan []byte
}

func NewUDPClient() Source {
	return &UDPClient{}
}

func (s UDPClient) GetType() SourceType {
	return UDPCLIENT
}

func (s UDPClient) Serve() {
	for {
		select {
		case conf := <-s.dial:
			if s.conn != nil {
				err := s.conn.Close()
				if err != nil {
					conf.c <- err
					continue
				}
			}
			addr, err := net.ResolveUDPAddr("udp", conf.addr)
			if err != nil {
				conf.c <- err
				continue
			}
			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				conf.c <- err
				continue
			}
			s.conn = conn
			go s.listen()
			conf.c <- nil
		case toSend := <-s.send:
			b := []byte(toSend.msg)
			n, _, err := s.conn.WriteMsgUDP(b, nil, nil)
			if err != nil {
				toSend.c <- err
				continue
			}
			if n < len(b) {
				toSend.c <- NewError("did not write whole message")
				continue
			}
			toSend.c <- nil
		case c := <-s.recv:
			c <- nil
		case c := <-s.quit:
			c <- nil
		}
	}
}

func (s UDPClient) listen() {
	// see http://stackoverflow.com/questions/1098897/what-is-the-largest-safe-udp-packet-size-on-the-internet
	b := make([]byte, 512)
	for {
		n, _, _, _, err := s.conn.ReadMsgUDP(b, nil)
		if err != nil {
			log.Fatal(err)
		}
		s.fromUDPClient <- b[:n]
	}
}

func (s UDPClient) ReceiveMessage(i chan Interrupt) ([]byte, Interrupt, error) {
	select {
	case msg, ok := <-s.fromUDPClient:
		if !ok {
			return nil, nil, errors.New("UDPClient connection has closed")
		}
		return msg, nil, nil
	case f := <-i:
		return nil, f, nil
	}

}
