package core

import (
	"log"
	"sync"
)

func ValueStore() SourceSpec {
	return SourceSpec{
		Name: "value",
		Type: VALUE_PRIMITIVE,
		New:  NewValue,
	}
}

type Value struct {
	value    interface{}
	quit     chan bool
	isLocked bool
	mutex    sync.Mutex
}

func NewValue() Source {
	return &Value{
		value:    nil,
		isLocked: false,
		quit:     make(chan bool),
	}
}

func (v Value) GetType() SourceType {
	return VALUE_PRIMITIVE
}

func (v *Value) Get() interface{} {
	return v.value
}

func (v *Value) Set(nv interface{}) error {
	v.value = nv
	return nil
}

func (v *Value) Lock() {
	v.mutex.Lock()
	v.isLocked = true
}

func (v *Value) Unlock() {
	if v.isLocked {
		v.mutex.Unlock()
	}
	v.isLocked = false
}

// ValueGet emits the value stored
func ValueGet() Spec {
	return Spec{
		Name: "valueGet",
		Inputs: []Pin{
			Pin{"trigger", ANY},
		},
		Outputs: []Pin{
			Pin{"value", ANY},
		},
		Source: VALUE_PRIMITIVE,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			v := s.(*Value)
			out[0] = v.value
			return nil
		},
	}
}

// ValueSet sets the value stored
func ValueSet() Spec {
	return Spec{
		Name: "valueSet",
		Inputs: []Pin{
			Pin{"newValue", ANY},
		},
		Outputs: []Pin{
			Pin{"out", BOOLEAN},
		},
		Source: VALUE_PRIMITIVE,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			v := s.(*Value)
			v.value = in[0]
			out[0] = true
			return nil
		},
	}
}

// ValueLock locks a value, emitting a semaphore
func Lock() Spec {
	return Spec{
		Name: "lock",
		Inputs: []Pin{
			Pin{"trigger", ANY},
		},
		Outputs: []Pin{
			Pin{"locked", BOOLEAN},
		},
		Source: STORE,
		OnReset: func(in, out, internal MessageMap, s Source) {
			v, ok := s.(Store)
			if !ok {
				// this can happen during reset, and s is null. just ignore..
				return
			}
			v.Unlock()
		},
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			v := s.(*Value)
			lockChan := make(chan bool)
			go func() {
				log.Println("getting lock")
				v.Lock()
				log.Println("got lock")
				lockChan <- true
			}()
			select {
			case out[0] = <-lockChan:
				return nil
			case f := <-i:
				log.Println("waiting on lock interrupted!")
				return f
			}
		},
	}
}

// Unlock releases a lock on a store
func Unlock() Spec {
	return Spec{
		Name: "unlock",
		Inputs: []Pin{
			Pin{"trigger", ANY},
		},
		Outputs: []Pin{},
		Source:  STORE,
		Kernel: func(in, out, internal MessageMap, s Source, i chan Interrupt) Interrupt {
			v := s.(Store)
			log.Println("unlocking")
			v.Unlock()
			log.Println("unlocked")
			return nil
		},
	}
}
