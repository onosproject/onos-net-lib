// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package realm contains utilities for applications operating on specific control domain realms
package realm

import (
	"fmt"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/spf13/cobra"
)

const (
	// LabelFlag command option
	LabelFlag = "realm-label"
	// ValueFlag command option
	ValueFlag = "realm-value"
	// LabelDefault option default
	LabelDefault = "realm"
	// ValueDefault option default
	ValueDefault = "*"
)

// AddRealmFlags injects standard realm label/value flags to the given command
func AddRealmFlags(cmd *cobra.Command, name string) {
	cmd.Flags().String(LabelFlag, LabelDefault,
		fmt.Sprintf("label used to define the realm of devices over which the %s should operate", name))
	cmd.Flags().String(ValueFlag, ValueDefault,
		fmt.Sprintf("value of the realm label of devices over which the %s should operate", name))
}

// Options holds onos-topo label/value for querying devices in specific control domain realms
type Options struct {
	Label string
	Value string
}

// ExtractOptions extracts realm options from the specified command usage
func ExtractOptions(cmd *cobra.Command) *Options {
	realmLabel, _ := cmd.Flags().GetString(LabelFlag)
	realmValue, _ := cmd.Flags().GetString(ValueFlag)
	return &Options{Label: realmLabel, Value: realmValue}
}

// QueryFilter returns query filter for entities with the given realm label/value and the specified aspects
func (ro *Options) QueryFilter(aspects ...string) *topo.Filters {
	filters := &topo.Filters{ObjectTypes: []topo.Object_Type{topo.Object_ENTITY}}
	if ro.Value != ValueDefault {
		filters.LabelFilters = []*topo.Filter{{
			Filter: &topo.Filter_Equal_{Equal_: &topo.EqualFilter{Value: ro.Value}},
			Key:    ro.Label,
		}}
	}
	if len(aspects) > 0 {
		filters.WithAspects = aspects
	}
	return filters
}
