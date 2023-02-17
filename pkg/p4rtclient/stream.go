// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package p4rtclient

import (
	"context"
	"github.com/onosproject/onos-net-lib/pkg/p4utils"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// StreamClient p4runtime master stream client
type StreamClient interface {
	PerformMasterArbitration(ctx context.Context, role *p4api.Role) (*p4api.StreamMessageResponse_Arbitration, error)
}

type streamClient struct {
	p4runtimeClient p4api.P4RuntimeClient
	deviceID        uint64
}

func (s *streamClient) PerformMasterArbitration(ctx context.Context, role *p4api.Role) (*p4api.StreamMessageResponse_Arbitration, error) {
	electionID := p4utils.TimeBasedElectionID()
	channel, err := s.p4runtimeClient.StreamChannel(ctx)
	if err != nil {
		return nil, err
	}

	request := &p4api.StreamMessageRequest{
		Update: &p4api.StreamMessageRequest_Arbitration{Arbitration: &p4api.MasterArbitrationUpdate{
			DeviceId:   s.deviceID,
			ElectionId: electionID,
			Role:       role,
		}},
	}
	log.Infow("Sending master arbitration request", "request", request)
	err = channel.Send(request)
	if err != nil {
		return nil, err
	}

	for {
		in, err := channel.Recv()
		if err != nil {
			return nil, err
		}

		switch v := in.Update.(type) {
		case *p4api.StreamMessageResponse_Arbitration:
			log.Infow("Received arbitration response", "response", v)
			if err != nil {
				return nil, err
			}
			return v, nil
		}
	}

}

var _ StreamClient = &streamClient{}
