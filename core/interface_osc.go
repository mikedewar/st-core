package core

import "github.com/hypebeast/go-osc/osc"

func OSCClient() SourceSpec {
	return SourceSpec{
		Name: "OSCClient",
		Type: OSCCLIENT,
		New:  NewOSC,
	}
}

func (OSC OSC) GetType() SourceType {
	return OSCCLIENT
}

type OSCMsg struct {
	address  string
	argument interface{} // TODO think harder
	err      chan error
}

type OSC struct {
	quit   chan chan error
	conf   chan OSCConf
	toSend chan OSCMsg
}

type OSCConf struct {
	url  string
	port int
	err  chan error
}

func NewOSC() Source {
	OSC := &OSC{
		quit:   make(chan chan error),
		toSend: make(chan OSCMsg),
		conf:   make(chan OSCConf),
	}
	return OSC
}

func (s *OSC) Serve() {
	var client *osc.Client
	for {
		select {
		case m := <-s.conf:
			client = osc.NewClient(m.url, m.port)
			m.err <- nil
		case m := <-s.toSend:
			msg := osc.NewMessage(m.address)
			msg.Append(m.argument)
			if client == nil {
				m.err <- NewError("client has not connected")
				continue
			}
			m.err <- client.Send(msg)
		}
	}

}

func (OSC *OSC) Stop() {
}

func OSCClientConnect() Spec {
	return Spec{
		Name:    "OSCClientConnect",
		Inputs:  []Pin{Pin{"url", STRING}, Pin{"port", NUMBER}},
		Outputs: []Pin{Pin{"connected", BOOLEAN}},
		Source:  OSCCLIENT,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			OSC := s.(*OSC)
			url, ok := in[0].(string)
			if !ok {
				out[0] = NewError("OSCClientConnect requires string url")
				return nil
			}
			port, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("OSCClientConnect requires number port")
				return nil
			}
			errChan := make(chan error)
			OSC.conf <- OSCConf{url, int(port), errChan}
			err := <-errChan
			if err != nil {
				out[0] = err
				return nil
			}
			out[0] = true
			return nil
		},
	}
}

func OSCClientSend() Spec {
	return Spec{
		Name:    "OSCClientSend",
		Inputs:  []Pin{Pin{"address", STRING}, Pin{"arugment", ANY}},
		Outputs: []Pin{Pin{"sent", BOOLEAN}},
		Source:  OSCCLIENT,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			OSC := s.(*OSC)
			addr, ok := in[0].(string)
			if !ok {
				out[0] = NewError("OSCSend requires string address")
				return nil
			}
			arg := in[1]
			errChan := make(chan error)
			msg := OSCMsg{addr, arg, errChan}
			OSC.toSend <- msg
			err := <-errChan
			if err != nil {
				out[0] = err
				return nil
			}
			out[0] = true
			return nil
		},
	}
}
