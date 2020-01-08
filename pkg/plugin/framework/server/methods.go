package server

import (
	"context"
	"fmt"

	pb "github.com/kubemove/kubemove/pkg/plugin/framework/proto"
	"github.com/pkg/errors"
)

func (s *server) RegisterPlugin(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	res := &pb.Response{}

	if len(req.Address) == 0 || len(req.Name) == 0 {
		fmt.Printf("Request has insufficient information\n")
		return res, errors.New("Request has insufficient information")
	}

	if _, ok := s.get(req.Name); ok {
		fmt.Printf("Plugin with name %s is already registered\n", req.Name)
		return res, errors.Errorf("Plugin with the same name is already registered")
	}

	p, err := newPlugin(req.Name, req.Address)
	if err != nil {
		fmt.Printf("Failed to create new plugin resources.. %v\n", err)
		return res, errors.Wrapf(err, "Failed to create new plugin resources")
	}

	p.path = s.cb(p.conn)

	if ok := s.Add(p); !ok {
		fmt.Printf("Failed to register new plugin..\n")
		return res, errors.New("Failed to register new plugin")
	}

	fmt.Printf("got req %v\n", req)
	return res, nil
}

func (s *server) get(name string) (plugin, bool) {
	s.RLock()
	defer s.RUnlock()
	p, ok := s.Plugin[name]
	return p, ok
}

func (s *server) Add(p *plugin) bool {
	if p == nil || len(p.name) == 0 {
		return false
	}

	name := p.name
	if _, ok := s.get(name); ok {
		return false
	}

	s.Lock()
	defer s.Unlock()
	s.Plugin[name] = *p
	return true
}

func (s *server) List() []string {
	var list []string
	s.RLock()
	defer s.RUnlock()
	for k := range s.Plugin {
		list = append(list, k)
	}
	return list
}

func (s *server) Get(plugin string, wait bool) interface{} {
	p, ok := s.get(plugin)
	if ok {
		//TODO
		_ = wait
		if !p.busy {
			p.busy = true
			return p.path
		}
		return nil
	}
	return nil
}

func (s *server) Put(plugin string) bool {
	p, ok := s.get(plugin)
	if ok {
		//TODO plugin lock
		p.busy = false
	}

	return true
}
