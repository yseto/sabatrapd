package main

import (
	"fmt"
	"strings"

	"github.com/sleepinggenius2/gosmi"
	"github.com/sleepinggenius2/gosmi/types"
)

type SMI struct {
	Modules []string
	Paths   []string
}

func (s *SMI) Init() error {
	gosmi.Init()

	for _, path := range s.Paths {
		gosmi.AppendPath(path)
	}
	for i, module := range s.Modules {
		moduleName, err := gosmi.LoadModule(module)
		if err != nil {
			return err
		}
		s.Modules[i] = moduleName
	}
	return nil
}

func (s *SMI) Close() {
	gosmi.Exit()
}

type Node struct {
	Readable string
	Node     gosmi.SmiNode
}

func (s *SMI) FromOID(oid string) (*Node, error) {
	var node gosmi.SmiNode
	var err error
	if (oid[0] >= '0' && oid[0] <= '9') || oid[0] == '.' {
		node, err = gosmi.GetNodeByOID(types.OidMustFromString(oid))
	} else {
		node, err = gosmi.GetNode(oid)
	}
	if err != nil {
		return nil, err
	}

	subtree := node.GetSubtree()

	if len(subtree) != 1 {
		return nil, fmt.Errorf("mismatch oid : %q, len : %d", oid, len(subtree))
	}

	if !subtree[0].Oid.ParentOf(types.OidMustFromString(oid)) {
		return nil, fmt.Errorf("mismatch oid. : %q", oid)
	}

	// readable
	readable := strings.Replace(
		types.OidMustFromString(oid).String(),
		subtree[0].RenderNumeric(),
		subtree[0].RenderQualified(),
		1,
	)

	return &Node{Readable: readable, Node: subtree[0]}, nil
}
