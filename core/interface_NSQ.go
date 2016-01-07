package core

import (
	"errors"
	"log"

	"github.com/bitly/go-nsq"
)

func NSQInterface() SourceSpec {
	return SourceSpec{
		Name: "NSQ",
		Type: NSQCLIENT,
		New:  NewNSQ,
	}
}

type NSQ struct {
	connectChan chan NSQConf
	sendChan    chan NSQMsg
	fromNSQ     chan string
	subscribe   chan chan string
	unsubscribe chan chan string
	subscribers map[chan string]struct{}
	quit        chan chan error
}

type NSQConf struct {
	conf       *nsq.Config
	topic      string
	channel    string
	lookupAddr string
	errChan    chan *stcoreError
}

type NSQMsg struct {
	msg     string
	errChan chan *stcoreError
}

func (s NSQ) GetType() SourceType {
	return NSQCLIENT
}

func NewNSQ() Source {
	return &NSQ{
		quit:        make(chan chan error),
		connectChan: make(chan NSQConf),
		sendChan:    make(chan NSQMsg),
		fromNSQ:     make(chan string),
		subscribe:   make(chan chan string),
		unsubscribe: make(chan chan string),
		subscribers: make(map[chan string]struct{}),
	}
}

/* config

*/

func (s NSQ) Serve() {
	var reader *nsq.Consumer
	for {
		select {
		case conf := <-s.connectChan:
			reader, err := nsq.NewConsumer(conf.topic, conf.channel, conf.conf)
			if err != nil {
				select {
				case conf.errChan <- NewError("NSQ failed to create Consumer with error:" + err.Error()):
				default:
				}
			}
			reader.AddHandler(s)
			err = reader.ConnectToNSQLookupd(conf.lookupAddr)
			if err != nil {
				select {
				case conf.errChan <- NewError("NSQ connect failed with:" + err.Error()):
				default:
				}
			}
		case _ = <-s.sendChan:
		case msg := <-s.fromNSQ:
			for c, _ := range s.subscribers {
				c <- msg
			}
		case c := <-s.subscribe:
			s.subscribers[c] = struct{}{}
		case c := <-s.unsubscribe:
			delete(s.subscribers, c)
		case <-s.quit:
			reader.Stop()
			<-reader.StopChan // this blocks until the reader is definitely dead
			// TODO have some sort of timeout here and return with error maybe?
			// don't forget the object coming through s.quite is an option error channel
		}
	}
}

func (s NSQ) HandleMessage(message *nsq.Message) error {
	s.fromNSQ <- string(message.Body)
	return nil
}

func (s NSQ) ReceiveMessage(i chan Interrupt) (string, Interrupt, error) {
	// receives message
	c := make(chan string, 10)
	s.subscribe <- c
	select {
	case msg, ok := <-c:
		if !ok {
			return "", nil, errors.New("NSQ connection has closed")
		}
		s.unsubscribe <- c
		return msg, nil, nil
	case f := <-i:
		s.unsubscribe <- c
		return "", f, nil
	}
}

func (s NSQ) Stop() {
	m := make(chan error)
	s.quit <- m
	// block until closed
	err := <-m
	if err != nil {
		log.Fatal(err)
	}
}

func NSQConnect() Spec {
	return Spec{
		Name:    "NSQConnect",
		Outputs: []Pin{Pin{"connected", BOOLEAN}},
		Inputs:  []Pin{Pin{"topic", STRING}, Pin{"channel", STRING}, Pin{"lookupAddr", STRING}, Pin{"maxInFlight", NUMBER}},
		Source:  NSQCLIENT,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {

			topic, ok := in[0].(string)
			if !ok {
				out[0] = NewError("NSQConnect requries string topic")
				return nil
			}
			channel, ok := in[1].(string)
			if !ok {
				out[0] = NewError("NSQConnect requries string channel")
				return nil
			}
			lookupAddr, ok := in[2].(string)
			if !ok {
				out[0] = NewError("NSQConnect requries string lookupAddr")
				return nil
			}
			maxInFlight, ok := in[3].(float64)
			if !ok {
				out[0] = NewError("NSQConnect requries number lookupAddr")
				return nil
			}

			conf := nsq.NewConfig()
			conf.MaxInFlight = int(maxInFlight)

			nsq, ok := s.(*NSQ)
			if !ok {
				log.Fatal("could not assert source is NSQ")
			}

			errChan := make(chan *stcoreError)

			connParams := NSQConf{
				conf:       conf,
				topic:      topic,
				channel:    channel,
				lookupAddr: lookupAddr,
				errChan:    errChan,
			}

			nsq.connectChan <- connParams

			// block on connect
			select {
			case err := <-errChan:
				if err != nil {
					out[0] = err
					return nil
				}
				out[0] = true
				return nil
			case f := <-i:
				return f
			}

		},
	}
}

// NSQRecieve receives messages from the NSQ system.
//
// OutPin 0: received message
func NSQReceive() Spec {
	return Spec{
		Name: "NSQReceive",
		Outputs: []Pin{
			Pin{"out", STRING},
		},
		Source: NSQCLIENT,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			nsq := s.(*NSQ)
			msg, f, err := nsq.ReceiveMessage(i)
			if err != nil {
				out[0] = err
				return nil
			}
			if f != nil {
				return f
			}
			out[0] = string(msg)
			return nil
		},
	}
}
