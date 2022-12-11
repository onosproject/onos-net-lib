// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package configtree

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	utils "github.com/onosproject/onos-net-lib/pkg/gnmiutils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"sync"
)

// Configurable provides an abstraction of a config tree-based configuration mechanism
type Configurable interface {
	// RefreshConfig refreshes the config tree state from any relevant external source state
	RefreshConfig()

	// UpdateConfig updates external target state using the config tree state.
	UpdateConfig()

	// ProcessConfigGet provides internals of a gNMI get request
	ProcessConfigGet(prefix *gnmi.Path, paths []*gnmi.Path) ([]*gnmi.Notification, error)

	// ProcessConfigSet provides internals of a gNMI set request
	ProcessConfigSet(prefix *gnmi.Path, updates []*gnmi.Update, replacements []*gnmi.Update, deletes []*gnmi.Path) ([]*gnmi.UpdateResult, error)

	// AddSubscribeResponder adds the given subscribe responder to the specified device
	AddSubscribeResponder(responder *SubscribeResponder)

	// RemoveSubscribeResponder removes the specified subscribe responder from the specified device
	RemoveSubscribeResponder(responder *SubscribeResponder)

	// GetSubscribeResponders returns list of currently registered subscribe responders
	GetSubscribeResponders() []SubscribeResponder
}

// SubscribeResponder is an abstraction for sending SubscribeResponse messages to controllers
type SubscribeResponder interface {
	// GetClientAddress returns the peer connection address
	GetClientAddress() string

	// TODO: move simapi.Connection to a neutral place and use that instead of a plain string above

	// Send queues up the specified response to asynchronously sends on the backing stream
	Send(response *gnmi.SubscribeResponse)
}

// GNMIConfigurable provides a base implementation of a gNMI server backed by a config tree
type GNMIConfigurable struct {
	Configurable
	root                *Node
	lock                sync.RWMutex
	subscribeResponders []SubscribeResponder
}

// NewGNMIConfigurable creates a new gNMI configurable backed by the specified config tree root.
func NewGNMIConfigurable(root *Node) *GNMIConfigurable {
	return &GNMIConfigurable{
		root:                root,
		subscribeResponders: make([]SubscribeResponder, 0, 4),
	}
}

// Root returns the root node of the backing config tree
func (c *GNMIConfigurable) Root() *Node {
	return c.root
}

// ProcessConfigGet provides internals of a gNMI get request
func (c *GNMIConfigurable) ProcessConfigGet(prefix *gnmi.Path, paths []*gnmi.Path) ([]*gnmi.Notification, error) {
	c.RefreshConfig()

	notifications := make([]*gnmi.Notification, 0, len(paths))
	rootNode := c.root
	if prefix != nil {
		ps := utils.ToString(prefix)
		if rootNode = rootNode.GetPath(ps); rootNode == nil {
			return nil, errors.NewInvalid("node with given prefix %s not found", ps)
		}
	}

	for _, path := range paths {
		nodes := rootNode.FindAll(utils.ToString(path))
		if len(nodes) > 0 {
			notifications = append(notifications, toNotification(prefix, nodes))
		}
	}

	// TODO: implement proper error handling
	return notifications, nil
}

// Creates a notification message from the specified nodes
func toNotification(prefix *gnmi.Path, nodes []*Node) *gnmi.Notification {
	updates := make([]*gnmi.Update, 0, len(nodes))
	for _, node := range nodes {
		updates = append(updates, toUpdate(node))
	}
	return &gnmi.Notification{
		Timestamp: 0,
		Prefix:    prefix,
		Update:    updates,
	}
}

// ProcessConfigSet provides internals of a gNMI set request
func (c *GNMIConfigurable) ProcessConfigSet(prefix *gnmi.Path, updates []*gnmi.Update, replacements []*gnmi.Update, deletes []*gnmi.Path) ([]*gnmi.UpdateResult, error) {
	opCount := len(updates) + len(replacements) + len(deletes)
	if opCount < 1 {
		return nil, errors.Status(errors.NewInvalid("no updates, replace or deletes")).Err()
	}
	results := make([]*gnmi.UpdateResult, 0, opCount)

	rootNode := c.root
	if prefix != nil {
		ps := utils.ToString(prefix)
		if rootNode = rootNode.GetPath(ps); rootNode == nil {
			return nil, errors.NewInvalid("node with given prefix %s not found", ps)
		}
	}

	for _, path := range deletes {
		rootNode.DeletePath(utils.ToString(path))
	}

	for _, update := range replacements {
		rootNode.ReplacePath(utils.ToString(update.Path), update.Val)
	}

	for _, update := range updates {
		rootNode.AddPath(utils.ToString(update.Path), update.Val)
	}

	// TODO: Implement proper result error reporting
	c.UpdateConfig()
	return results, nil
}

// Creates an update message from the specified node
func toUpdate(node *Node) *gnmi.Update {
	return &gnmi.Update{
		Path:       utils.ToPath(node.Path()),
		Val:        node.Value(),
		Duplicates: 0,
	}
}

// AddSubscribeResponder adds the given subscribe responder to the specified device
func (c *GNMIConfigurable) AddSubscribeResponder(responder SubscribeResponder) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.subscribeResponders = append(c.subscribeResponders, responder)
}

// RemoveSubscribeResponder removes the specified subscribe responder from the specified device
func (c *GNMIConfigurable) RemoveSubscribeResponder(responder SubscribeResponder) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for i, r := range c.subscribeResponders {
		if r == responder {
			c.subscribeResponders = append(c.subscribeResponders[:i], c.subscribeResponders[i+1:]...)
			return
		}
	}
}
