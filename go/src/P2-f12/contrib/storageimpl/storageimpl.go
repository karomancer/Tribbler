package storageimpl

// The internal implementation of your storage server.
// Note:  This file does *not* provide a 'main' interface.  It is used
// by the 'storageserver' main function we have provided you, which
// will call storageimpl.NewStorageserver(...).
//
// Must implemement NewStorageserver and the RPC-able functions
// defined in the storagerpc StorageInterface interface.


import (
	"P2-f12/official/storageproto"
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
)

type Storageserver struct {
}

func reallySeedTheDamnRNG() {
	randint, _ := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	rand.Seed( randint.Int64())
}

func NewStorageserver(master string, numnodes int, portnum int, nodeid uint32) *Storageserver {
	return &Storageserver{}
}

// Non-master servers to the master
func (ss *Storageserver) RegisterServer(args *storageproto.RegisterArgs, reply *storageproto.RegisterReply) error {
	return nil
}

func (ss *Storageserver) GetServers(args *storageproto.GetServersArgs, reply *storageproto.RegisterReply) error {
	return nil
}

// RPC-able interfaces, bridged via StorageRPC.
// These should do something! :-)

func (ss *Storageserver) Get(args *storageproto.GetArgs, reply *storageproto.GetReply) error {
	return nil
}

func (ss *Storageserver) GetList(args *storageproto.GetArgs, reply *storageproto.GetListReply) error {
	return nil
}

func (ss *Storageserver) Put(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	return nil
}

func (ss *Storageserver) AppendToList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
 	return nil
}

func (ss *Storageserver) RemoveFromList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
 	return nil
}
