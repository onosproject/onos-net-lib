// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package configtree

import (
	"fmt"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestDeviceConfig is used as a playground to validate the creation of device gNMI configtree.
func TestDeviceConfig(t *testing.T) {
	rootNode := newSwitchTestConfig()
	assert.NotNil(t, rootNode.Get("interfaces", nil))

	node := rootNode.GetPath("interfaces/interface[name=5]/state/id")
	assert.NotNil(t, node.Value().GetIntVal())
	assert.Equal(t, "id", node.Name())
	assert.Equal(t, uint64(1029), node.Value().GetUintVal())

	nodes := rootNode.FindAll("interfaces/interface[name=7]")
	assert.Len(t, nodes, 20)

	nodes = rootNode.FindAll("interfaces/interface[name=7]/state")
	assert.Len(t, nodes, 18)

	nodes = rootNode.FindAll("interfaces/interface[name=7]/state/counters")
	assert.Len(t, nodes, 14)

	nodes = rootNode.FindAll("interfaces/interface[name=7]/state/ifindex")
	assert.Len(t, nodes, 1)

	nodes = rootNode.FindAll("interfaces/interface[name=...]")
	assert.Len(t, nodes, 8*20)

	nodes = rootNode.FindAll("interfaces/interface[name=...]/state")
	assert.Len(t, nodes, 8*18)

	nodes = rootNode.FindAll("interfaces/interface[name=...]/state/ifindex")
	assert.Len(t, nodes, 8)

	nodes = rootNode.FindAll("interfaces/interface[name=...]/state/counters")
	assert.Len(t, nodes, 8*14)

	node = rootNode.GetPath("interfaces/interface[name=2]/state/counters")
	assert.NotNil(t, node)
	node = rootNode.DeletePath("interfaces/interface[name=2]/state/counters")
	assert.NotNil(t, node)
	node = rootNode.GetPath("interfaces/interface[name=2]/state/counters")
	assert.Nil(t, node)
}

func newSwitchTestConfig() *Node {
	rootNode := NewRoot()

	interfacesNode := rootNode.Add("interfaces", nil, nil)
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("%d", i)
		interfaceNode := interfacesNode.Add("interface", map[string]string{"name": name}, nil)

		interfaceNode.AddPath("state/ifindex",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: uint64(i)}})
		interfaceNode.AddPath("state/id",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: uint64(1024 + i)}})

		portStatus := "UP"
		if i == 4 || i == 7 {
			portStatus = "DOWN"
		}
		interfaceNode.AddPath("state/oper-status",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: portStatus}})

		interfaceNode.AddPath("state/last-change",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: 0}})

		interfaceNode.AddPath("configtree/enabled",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_BoolVal{BoolVal: portStatus == "UP"}})
		interfaceNode.AddPath("ethernet/configtree/port-speed",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "100Gbps"}})

		addCounters(interfaceNode)
	}

	return rootNode
}

var supportedCounters = []string{
	"in-octets",
	"out-octets",
	"in-discards",
	"in-fcs-errors",
	"out-discards",
	"in-errors",
	"out-errors",
	"in-unicast-pkts",
	"in-broadcast-pkts",
	"in-multicast-pkts",
	"in-unknown-protos",
	"out-unicast-pkts",
	"out-broadcast-pkts",
	"out-multicast-pkts",
}

func addCounters(node *Node) {
	countersNode := node.AddPath("state/counters", nil)
	for _, counter := range supportedCounters {
		countersNode.Add(counter, nil, &gnmi.TypedValue{Value: &gnmi.TypedValue_IntVal{IntVal: 0}})
	}
}
