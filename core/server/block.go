package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/nikhan/go-fetch"
	"github.com/nytlabs/st-core/core"
	"github.com/thejerf/suture"
)

type BlockLedger struct {
	Label       string              `json:"label"`
	Type        string              `json:"type"`
	Id          int                 `json:"id"`
	Block       *core.Block         `json:"-"`
	Parent      *Group              `json:"-"`
	Token       suture.ServiceToken `json:"-"`
	Composition int                 `json:"composition,omitempty"`
	Inputs      []BlockLedgerInput  `json:"inputs"`
	Outputs     []core.Output       `json:"outputs"`
}

func (bl *BlockLedger) GetID() int {
	return bl.Id
}

func (bl *BlockLedger) GetParent() *Group {
	return bl.Parent
}

func (bl *BlockLedger) SetParent(group *Group) {
	bl.Parent = group
}

type BlockLedgerInput struct {
	Name  string            `json:"name"`
	Type  string            `json:"type"`
	Value interface{}       `json:"value"`
	C     chan core.Message `json:"-"`
}

func (s *Server) ListBlocks() []BlockLedger {
	blocks := []BlockLedger{}
	for _, b := range s.blocks {
		blocks = append(blocks, *b)
	}
	return blocks
}

func (s *Server) BlockIndexHandler(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(s.ListBlocks()); err != nil {
		panic(err)
	}
}

func (s *Server) BlockHandler(w http.ResponseWriter, r *http.Request) {
}
func (s *Server) BlockModifyPositionHandler(w http.ResponseWriter, r *http.Request) {
}

// CreateBlockHandler responds to a POST request to instantiate a new block and add it to the Server.
func (s *Server) BlockCreateHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"could not read request body"})
		return
	}

	var m BlockLedger
	err = json.Unmarshal(body, &m)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"no ID supplied"})
		return
	}

	s.Lock()
	defer s.Unlock()

	blockSpec, ok := s.library[m.Type]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"spec not found"})
		return
	}

	m.Id = s.GetNextID()
	m.Block = core.NewBlock(blockSpec)
	m.Token = s.supervisor.Add(m.Block)
	is := m.Block.GetRoutes()

	// may want to move this into actual block someday
	inputs := make([]BlockLedgerInput, len(is), len(is))
	for i, v := range is {
		if q, ok := v.Value.(*fetch.Query); ok {
			inputs[i] = BlockLedgerInput{
				Name:  v.Name,
				Type:  "fetch",
				Value: q.String(),
				C:     v.C,
			}
		} else {
			inputs[i] = BlockLedgerInput{
				Name:  v.Name,
				Type:  "const",
				Value: v.Value,
				C:     v.C,
			}
		}
	}

	m.Inputs = inputs
	m.Outputs = m.Block.GetOutputs()
	s.blocks[m.Id] = &m
	s.websocketBroadcast(Update{Action: CREATE, Type: BLOCK, Data: m})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) BlockModifyRouteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ids, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"no ID supplied"})
		return
	}

	id, err := strconv.Atoi(ids)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{err.Error()})
		return
	}

	routes, ok := vars["index"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"no route index supplied"})
		return
	}

	route, err := strconv.Atoi(routes)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{err.Error()})
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"could not read request body"})
		return
	}

	s.Lock()
	defer s.Unlock()

	b, ok := s.blocks[id]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"block not found"})
		return
	}

	var v BlockLedgerInput
	err = json.Unmarshal(body, &v)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"could not unmarshal value"})
		return
	}

	// again maybe this type should be native to block under core.
	var m interface{}
	switch v.Type {
	case "fetch":
		queryString, ok := v.Value.(string)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, Error{"fetch is not string"})
			return
		}

		fo, err := fetch.Parse(queryString)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, Error{err.Error()})
			return
		}

		m = fo
	case "const":
		m = v.Value
	default:
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"no value or query specified"})
		return
	}

	err = b.Block.SetRoute(core.RouteID(route), m)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{err.Error()})
		return
	}

	s.blocks[id].Inputs[route].Type = v.Type
	s.blocks[id].Inputs[route].Value = m

	update := struct {
		BlockLedgerInput
		Id    int `json:"id"`
		input int `json:"input"`
	}{
		v, id, route,
	}

	s.websocketBroadcast(Update{Action: UPDATE, Type: BLOCK, Data: update})
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) BlockModifyNameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ids, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"no ID supplied"})
		return
	}

	id, err := strconv.Atoi(ids)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{err.Error()})
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"could not read request body"})
		return
	}

	s.Lock()
	defer s.Unlock()

	_, ok = s.blocks[id]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"block not found"})
		return
	}

	var label string
	err = json.Unmarshal(body, &label)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"could not unmarshal value"})
		return
	}

	s.blocks[id].Label = label

	update := struct {
		Id    int    `json:"id"`
		Label string `json:"label"`
	}{
		id, label,
	}

	s.websocketBroadcast(Update{Action: UPDATE, Type: BLOCK, Data: update})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) DeleteBlock(id int) error {
	b, ok := s.blocks[id]
	if !ok {
		return errors.New("block not found")
	}

	deleteSet := make(map[int]struct{})

	// build a set of connections that we may need to delete
	// we need to panic here because if any error is thrown we are in huge trouble
	// any panic indicates that our server connection ledger is no longer true
	for _, c := range s.connections {
		if c.Target.Id == id || c.Source.Id == id {
			deleteSet[c.Id] = struct{}{}
		}
		/*if c.Target.Id == id {
			route, err := b.Block.GetRoute(core.RouteID(c.Target.Route))
			if err != nil {
				panic(err)
			}
			err = s.blocks[c.Source.Id].Block.Disconnect(core.RouteID(c.Source.Route), route.C)
			if err != nil {
				panic(err)
			}
			deleteSet[c.Id] = struct{}{}
		}
		if c.Source.Id == id {
			route, err := s.blocks[c.Target.Id].Block.GetRoute(core.RouteID(c.Target.Route))
			if err != nil {
				panic(err)
			}
			err = b.Block.Disconnect(core.RouteID(c.Source.Route), route.C)
			if err != nil {
				panic(err)
			}
			deleteSet[c.Id] = struct{}{}
		}*/
	}

	// delete the connections that involve this block
	for k, _ := range deleteSet {
		s.DeleteConnection(k)
		//s.websocketBroadcast(Update{Action: DELETE, Type: CONNECTION, Data: s.connections[k]})
		//delete(s.connections, k)
	}

	// stop and delete the block
	s.supervisor.Remove(b.Token)
	s.websocketBroadcast(Update{Action: DELETE, Type: BLOCK, Data: s.blocks[id]})
	delete(s.blocks, id)
	return nil
}

func (s *Server) BlockDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ids, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{"no ID supplied"})
		return
	}

	id, err := strconv.Atoi(ids)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{err.Error()})
		return
	}

	s.Lock()
	defer s.Unlock()

	err = s.DeleteBlock(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, Error{err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
