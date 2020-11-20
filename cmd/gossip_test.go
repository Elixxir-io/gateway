///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gateway/storage"
	"gitlab.com/elixxir/primitives/rateLimiting"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/gossip"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"os"
	"testing"
	"time"
)

// Happy path
func TestInstance_GossipReceive_RateLimit(t *testing.T) {
	gatewayInstance.InitRateLimitGossip()
	defer gatewayInstance.KillRateLimiter()
	var err error

	// Create a fake round info
	ri := &pb.RoundInfo{
		ID:       10,
		UpdateID: 10,
	}

	// Sign the round info with the mock permissioning private key
	err = signRoundInfo(ri)
	if err != nil {
		t.Errorf("Error signing round info: %s", err)
	}

	// Build a test batch
	batch := &pb.Batch{
		Slots: make([]*pb.Slot, 10),
		Round: ri,
	}

	for i := 0; i < len(batch.Slots); i++ {
		senderId := id.NewIdFromString(fmt.Sprintf("%d", i), id.User, t)
		batch.Slots[i] = &pb.Slot{SenderID: senderId.Marshal()}
	}

	// Build a test gossip message
	gossipMsg := &gossip.GossipMsg{}
	gossipMsg.Payload, err = buildGossipPayloadRateLimit(batch)
	if err != nil {
		t.Errorf("Unable to build gossip payload: %+v", err)
	}

	// Test the gossipRateLimitReceive function
	err = gatewayInstance.gossipRateLimitReceive(gossipMsg)
	if err != nil {
		t.Errorf("Unable to receive gossip message: %+v", err)
	}

	// Ensure the buckets were populated
	for _, slot := range batch.Slots {
		senderId, err := id.Unmarshal(slot.GetSenderID())
		if err != nil {
			t.Errorf("Could not unmarshal sender ID: %+v", err)
		}
		bucket := gatewayInstance.rateLimit.LookupBucket(senderId.String())
		if bucket.Remaining() == 0 {
			t.Errorf("Failed to add to leaky bucket for sender %s", senderId.String())
		}
	}
}

// Happy path
func TestInstance_GossipVerify(t *testing.T) {
	//Build the gateway instance
	params := Params{
		NodeAddress:           NODE_ADDRESS,
		ServerCertPath:        testkeys.GetNodeCertPath(),
		CertPath:              testkeys.GetGatewayCertPath(),
		MessageTimeout:        10 * time.Minute,
		KeyPath:               testkeys.GetGatewayKeyPath(),
		PermissioningCertPath: testkeys.GetNodeCertPath(),
		knownRoundsPath:       "kr.json",
	}

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(params.knownRoundsPath)
		if err != nil {
			t.Fatalf("Error deleting test file: %v", err)
		}
	}()

	params.rateLimitParams = &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokens,
		LeakDuration: leakDuration,
		PollDuration: pollDuration,
		BucketMaxAge: bucketMaxAge,
	}

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(params.knownRoundsPath)
		if err != nil {
			t.Fatalf("Error deleting test file: %v", err)
		}
	}()

	gw := NewGatewayInstance(params)
	p := large.NewIntFromString(prime, 16)
	g := large.NewIntFromString(generator, 16)
	grp2 := cyclic.NewGroup(p, g)

	gw.Comms = gateway.StartGateway(&id.TempGateway, "0.0.0.0:11690", gw,
		gatewayCert, gatewayKey, gossip.DefaultManagerFlags())

	testNDF, _, _ := ndf.DecodeNDF(ExampleJSON + "\n" + ExampleSignature)

	var err error
	gw.NetInf, err = network.NewInstanceTesting(gw.Comms.ProtoComms, testNDF, testNDF, grp2, grp2, t)
	if err != nil {
		t.Errorf("NewInstanceTesting encountered an error: %+v", err)
	}

	gw.InitRateLimitGossip()
	defer gw.KillRateLimiter()

	// Add permissioning as a host
	pub := testkeys.LoadFromPath(testkeys.GetNodeCertPath())
	_, err = gw.Comms.AddHost(&id.Permissioning,
		"0.0.0.0:4200", pub, connect.GetDefaultHostParams())

	originId := id.NewIdFromString("test", id.Gateway, t)

	// Build a mock node ID for a topology
	idCopy := originId.DeepCopy()
	idCopy.SetType(id.Node)
	topology := [][]byte{idCopy.Bytes()}

	// Create a fake round info to store
	ri := &pb.RoundInfo{
		ID:       10,
		UpdateID: 10,
		Topology: topology,
	}

	// Sign the round info with the mock permissioning private key
	err = signRoundInfo(ri)
	if err != nil {
		t.Errorf("Error signing round info: %s", err)
	}

	// Insert the mock round into the network instance
	err = gw.NetInf.RoundUpdate(ri)
	if err != nil {
		t.Errorf("Could not place mock round: %v", err)
	}

	// ----------- Rate Limit Check ---------------------

	// Build the mock message
	payloadMsgRateLimit := &pb.BatchSenders{
		SenderIds: topology,
		RoundID:   10,
	}

	// Marshal the payload for the gossip message
	payload, err := proto.Marshal(payloadMsgRateLimit)
	if err != nil {
		t.Errorf("Could not marshal mock message: %s", err)
	}

	// Build a test gossip message
	gossipMsg := &gossip.GossipMsg{
		Tag:     RateLimitGossip,
		Origin:  originId.Marshal(),
		Payload: payload,
	}
	gossipMsg.Signature, err = buildGossipSignature(gossipMsg, gw.Comms.GetPrivateKey())

	// Set up origin host
	_, err = gw.Comms.AddHost(originId, "", gatewayCert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to add test host: %+v", err)
	}

	// Test the gossipVerify function
	err = gw.gossipVerify(gossipMsg, nil)
	if err != nil {
		t.Errorf("Unable to verify gossip message: %+v", err)
	}

	// ----------- Bloom Filter Check ---------------------
	// Build the mock message
	payloadMsgBloom := &pb.Recipients{
		RecipientIds: topology,
		RoundID:      10,
	}

	// Marshal the payload for the gossip message
	payload, err = proto.Marshal(payloadMsgBloom)
	if err != nil {
		t.Errorf("Could not marshal mock message: %s", err)
	}

	// Build a test gossip message
	gossipMsg = &gossip.GossipMsg{
		Tag:     BloomFilterGossip,
		Origin:  originId.Marshal(),
		Payload: payload,
	}
	gossipMsg.Signature, err = buildGossipSignature(gossipMsg, gw.Comms.GetPrivateKey())

	// Test the gossipVerify function
	err = gw.gossipVerify(gossipMsg, nil)
	if err != nil {
		t.Errorf("Unable to verify gossip message: %+v", err)
	}

}

// Happy path
func TestInstance_StartPeersThread(t *testing.T) {
	gatewayInstance.addGateway = make(chan network.NodeGateway, gwChanLen)
	gatewayInstance.removeGateway = make(chan *id.ID, gwChanLen)
	gatewayInstance.InitRateLimitGossip()
	gatewayInstance.InitBloomGossip()
	defer gatewayInstance.KillRateLimiter()
	var err error

	// Prepare values and host
	gwId := id.NewIdFromString("test", id.Gateway, t)
	testSignal := network.NodeGateway{
		Gateway: ndf.Gateway{
			ID: gwId.Marshal(),
		},
	}
	_, err = gatewayInstance.Comms.AddHost(gwId, "0.0.0.0", gatewayCert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to add test host: %+v", err)
	}
	protocol, exists := gatewayInstance.Comms.Manager.Get(RateLimitGossip)
	if !exists {
		t.Errorf("Unable to get gossip protocol!")
		return
	}

	// Start the channel monitor
	gatewayInstance.StartPeersThread()

	// Send the add gateway signal
	gatewayInstance.addGateway <- testSignal


	// Test the add gateway signals
	// by attempting to remove the added gateway
	for i := 0; i < 5; i++ {
		err = protocol.RemoveGossipPeer(gwId)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Errorf("Unable to remove gossip peer: %+v", err)
	}

	// Now add a peer and send a a remove signal
	err = protocol.AddGossipPeer(gwId)
	if err != nil {
		t.Errorf("Unable to add gossip peer: %+v", err)
	}
	gatewayInstance.removeGateway <- gwId

	// Test the remove gateway signals
	// by attempting to remove a gateway that should have already been removed
	time.Sleep(100 * time.Millisecond)
	err = protocol.RemoveGossipPeer(gwId)
	if err == nil {
		t.Errorf("Expected failure to remove already-removed peer!")
	}
}

//
func TestInstance_GossipBatch(t *testing.T) {
	//Build the gateway instance
	params := Params{
		NodeAddress:           NODE_ADDRESS,
		ServerCertPath:        testkeys.GetNodeCertPath(),
		CertPath:              testkeys.GetGatewayCertPath(),
		MessageTimeout:        10 * time.Minute,
		KeyPath:               testkeys.GetGatewayKeyPath(),
		PermissioningCertPath: testkeys.GetNodeCertPath(),
		knownRoundsPath:       "kr.json",
	}

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(params.knownRoundsPath)
		if err != nil {
			t.Fatalf("Error deleting test file: %v", err)
		}
	}()

	params.rateLimitParams = &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokens,
		LeakDuration: leakDuration,
		PollDuration: pollDuration,
		BucketMaxAge: bucketMaxAge,
	}

	gw := NewGatewayInstance(params)
	p := large.NewIntFromString(prime, 16)
	g := large.NewIntFromString(generator, 16)
	grp2 := cyclic.NewGroup(p, g)
	addr := "0.0.0.0:6666"
	gw.Comms = gateway.StartGateway(&id.TempGateway, addr, gw,
		gatewayCert, gatewayKey, gossip.DefaultManagerFlags())

	testNDF, _, _ := ndf.DecodeNDF(ExampleJSON + "\n" + ExampleSignature)

	var err error
	gw.NetInf, err = network.NewInstanceTesting(gw.Comms.ProtoComms, testNDF, testNDF, grp2, grp2, t)
	if err != nil {
		t.Errorf("NewInstanceTesting encountered an error: %+v", err)
	}

	gw.InitRateLimitGossip()
	defer gw.KillRateLimiter()

	// Add permissioning as a host
	pub := testkeys.LoadFromPath(testkeys.GetNodeCertPath())
	_, err = gw.Comms.AddHost(&id.Permissioning,
		"0.0.0.0:4200", pub, connect.GetDefaultHostParams())

	// Init comms and host
	_, err = gw.Comms.AddHost(gw.Comms.Id, addr, gatewayCert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to add test host: %+v", err)
	}
	protocol, exists := gw.Comms.Manager.Get(RateLimitGossip)
	if !exists {
		t.Errorf("Unable to get gossip protocol!")
		return
	}
	err = protocol.AddGossipPeer(gw.Comms.Id)
	if err != nil {
		t.Errorf("Unable to add gossip peer: %+v", err)
	}

	// Build a mock node ID for a topology
	nodeID := gw.Comms.Id.DeepCopy()
	nodeID.SetType(id.Node)
	topology := [][]byte{nodeID.Bytes()}
	// Create a fake round info to store
	ri := &pb.RoundInfo{
		ID:       10,
		UpdateID: 10,
		Topology: topology,
	}

	// Sign the round info with the mock permissioning private key
	err = signRoundInfo(ri)
	if err != nil {
		t.Errorf("Error signing round info: %s", err)
	}

	// Insert the mock round into the network instance
	err = gw.NetInf.RoundUpdate(ri)
	if err != nil {
		t.Errorf("Could not place mock round: %v", err)
	}

	// Build a test batch
	batch := &pb.Batch{
		Round: ri,
		Slots: make([]*pb.Slot, 10),
	}
	for i := 0; i < len(batch.Slots); i++ {
		senderId := id.NewIdFromString(fmt.Sprintf("%d", i), id.User, t)
		batch.Slots[i] = &pb.Slot{SenderID: senderId.Marshal()}
	}

	// Send the gossip
	err = gw.GossipBatch(batch)
	if err != nil {
		t.Errorf("Unable to gossip: %+v", err)
	}

	// Verify the gossip was received
	testSenderId := id.NewIdFromString("0", id.User, t)
	if remaining := gw.rateLimit.LookupBucket(testSenderId.String()).Remaining(); remaining != 1 {
		t.Errorf("Expected to reduce remaining message count for test sender, got %d", remaining)
	}
}

func TestInstance_GossipBloom(t *testing.T) {
	//Build the gateway instance
	params := Params{
		NodeAddress:           NODE_ADDRESS,
		ServerCertPath:        testkeys.GetNodeCertPath(),
		CertPath:              testkeys.GetGatewayCertPath(),
		MessageTimeout:        10 * time.Minute,
		KeyPath:               testkeys.GetGatewayKeyPath(),
		PermissioningCertPath: testkeys.GetNodeCertPath(),
		knownRoundsPath:       "kr.json",
	}

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(params.knownRoundsPath)
		if err != nil {
			t.Fatalf("Error deleting test file: %v", err)
		}
	}()

	params.rateLimitParams = &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokens,
		LeakDuration: leakDuration,
		PollDuration: pollDuration,
		BucketMaxAge: bucketMaxAge,
	}

	gw := NewGatewayInstance(params)
	p := large.NewIntFromString(prime, 16)
	g := large.NewIntFromString(generator, 16)
	grp2 := cyclic.NewGroup(p, g)
	addr := "0.0.0.0:7777"
	gw.Comms = gateway.StartGateway(&id.TempGateway, addr, gw,
		gatewayCert, gatewayKey, gossip.DefaultManagerFlags())

	testNDF, _, _ := ndf.DecodeNDF(ExampleJSON + "\n" + ExampleSignature)

	var err error
	gw.NetInf, err = network.NewInstanceTesting(gw.Comms.ProtoComms, testNDF, testNDF, grp2, grp2, t)
	if err != nil {
		t.Errorf("NewInstanceTesting encountered an error: %+v", err)
	}

	rndId := uint64(10)

	gw.storage.InsertEpoch(id.Round(rndId))

	gw.InitBloomGossip()

	// Add permissioning as a host
	pub := testkeys.LoadFromPath(testkeys.GetNodeCertPath())
	_, err = gw.Comms.AddHost(&id.Permissioning,
		"0.0.0.0:4200", pub, connect.GetDefaultHostParams())

	// Init comms and host
	_, err = gw.Comms.AddHost(gw.Comms.Id, addr, gatewayCert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to add test host: %+v", err)
	}
	protocol, exists := gw.Comms.Manager.Get(BloomFilterGossip)
	if !exists {
		t.Errorf("Unable to get gossip protocol!")
		return
	}
	err = protocol.AddGossipPeer(gw.Comms.Id)
	if err != nil {
		t.Errorf("Unable to add gossip peer: %+v", err)
	}

	// Build a mock node ID for a topology
	nodeID := gw.Comms.Id.DeepCopy()
	nodeID.SetType(id.Node)
	topology := [][]byte{nodeID.Bytes()}
	// Create a fake round info to store
	ri := &pb.RoundInfo{
		ID:       rndId,
		UpdateID: 10,
		Topology: topology,
	}

	// Sign the round info with the mock permissioning private key
	err = signRoundInfo(ri)
	if err != nil {
		t.Errorf("Error signing round info: %s", err)
	}

	// Insert the mock round into the network instance
	err = gw.NetInf.RoundUpdate(ri)
	if err != nil {
		t.Errorf("Could not place mock round: %v", err)
	}

	clients := make(map[id.ID]interface{})
	for i := uint64(0); i < 10; i++ {
		tempId := id.NewIdFromUInt(i, id.User, t)
		clients[*tempId] = nil
	}

	// Insert the first five IDs as known clients
	i := 0
	for client := range clients{
		mockClient := &storage.Client{
			Id: client.Bytes(),
		}
		gw.storage.InsertClient(mockClient)
		i++
		if i==5{
			break
		}
	}

	// Send the gossip
	err = gw.GossipBloom(clients, id.Round(rndId))
	if err != nil {
		t.Errorf("Unable to gossip: %+v", err)
	}
	time.Sleep(1 * time.Second)

	i = 0
	for clientId  := range clients {
		// Check that the first five IDs are known clients, and thus
		// in the user bloom filter
		filters, err := gw.storage.GetBloomFilters(&clientId, id.Round(rndId))
		if err != nil || filters == nil {
			t.Errorf("Could not get a bloom filter for user %d with ID %s", i, clientId)
		}
		i++
	}
}

// Utility function which signs a round info message
func signRoundInfo(ri *pb.RoundInfo) error {
	privKeyFromFile := testkeys.LoadFromPath(testkeys.GetNodeKeyPath())

	pk, err := tls.LoadRSAPrivateKey(string(privKeyFromFile))
	if err != nil {
		return errors.Errorf("Couldn't load private key: %+v", err)
	}

	ourPrivateKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivateKey)
	return nil
}