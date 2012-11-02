package storageimpl

// The internal implementation of your storage server.
// Note:  This file does *not* provide a 'main' interface.  It is used
// by the 'storageserver' main function we have provided you, which
// will call storageimpl.NewStorageserver(...).
//
// Must implemement NewStorageserver and the RPC-able functions
// defined in the storagerpc StorageInterface interface.


//TO DO
//check each key if we are on the right server (rehash and check against our id, and other servers too since we have a list of all servers)
//leaseTimer
//other shits
//like figure out what a server is supposed to print and stuff when it joins and things

import (
	"P2-f12/official/storageproto"
	crand "crypto/rand"
	"math"
	"strconv"
	"math/big"
	"time"
	"net/rpc"
	"math/rand"
)

type Storageserver struct {
	nodeid uint32
	isMaster bool
	numNodes int
	nodeList []storageproto.Node
	nodeListM chan int
	nodeMap map[storageproto.Node]int
	nodeMapM chan int
	leaseMap map[string][]string //maps key to the clients who hold a lease on that key
	leaseMapM chan int

	listMap map[string][]string
	listMapM chan int

	valMap map[string]string
	valMapM chan int
}

func reallySeedTheDamnRNG() {
	randint, _ := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	rand.Seed( randint.Int64())
}

func NewStorageserver(master string, numnodes int, portnum int, nodeid uint32) *Storageserver {
	ss := &Storageserver{}

	//if no nodeid is provided, choose one randomly
	if nodeid == 0 {
		reallySeedTheDamnRNG()
		ss.nodeid = rand.Uint32()
	} else {
		//otherwise just take the one they gave you
		ss.nodeid = nodeid
	}

	if numnodes != 0 && master == "" {
		ss.isMaster = true
	} else {
		ss.isMaster = false
	}

	ss.nodeList = []storageproto.Node{}
	ss.nodeListM = make(chan int, 1)
	ss.nodeListM <- 1

	ss.nodeMap = make(map[storageproto.Node]int)
	ss.nodeMapM = make(chan int, 1)
	ss.nodeMapM <- 1

	ss.leaseMap = make(map[string][]string)
	ss.leaseMapM = make(chan int, 1)
	ss.leaseMapM <- 1

	ss.listMap = make(map[string][]string)
	ss.listMapM = make(chan int, 1)
	ss.listMapM <- 1

	ss.valMap = make(map[string]string)
	ss.valMapM = make(chan int, 1)
	ss.valMapM <- 1

	if ss.isMaster == false {
		//connect to the master server
		var masterClient *rpc.Client
		var err error
		for err != nil {
			//keep retrying until we can actually conenct
			//(Master may not have started yet)
			masterClient, err = rpc.DialHTTP("tcp", master)
			time.Sleep(time.Duration(3)*time.Second)	
		}

		//set up args for registering ourselves
		info := storageproto.Node{HostPort: ":" + strconv.Itoa(portnum), NodeID: nodeid}
		args := storageproto.RegisterArgs{ServerInfo: info}
		reply := storageproto.RegisterReply{}

		for err != nil || reply.Ready != true {
			//call register on the master node with our info as the args. Kinda weird
			err = masterClient.Call("StorageRPC.RegisterServer", args, reply)
			//keep retrying until all things are registered
			time.Sleep(time.Duration(3)*time.Second)	
		}

		//gotta still set up some other shits
		//like get list of servers from reply maybe?
		//spec is pretty vague...
		<- ss.nodeListM
		ss.nodeList = reply.Servers
		ss.nodeListM <- 1

		//non master doesn't keep a node map cause fuck you

	} else {
		//otherwise we are the master so we have to do some other shits
		//maybe these are the same shits as the other dudes got to do so maybe we don't gotta bother? WHAT WHO KNOWS

		//set up numNodes:
		if (numnodes == 0) {
			numnodes = 1
		}
		ss.numNodes = numnodes;
		<- ss.nodeListM
		//append self to nodeList and put self in map
		ss.nodeList = append(ss.nodeList, storageproto.Node{})//some shit here})
		ss.nodeListM <- 1
		<- ss.nodeMapM
		ss.nodeMap[storageproto.Node{}] = 0 //someshithere}] = 0
	}

	return ss
}

// Non-master servers to the master
func (ss *Storageserver) RegisterServer(args *storageproto.RegisterArgs, reply *storageproto.RegisterReply) error {
	//called on master by other servers
	//first check if that server is alreayd in our map
	<- ss.nodeMapM
	_, ok := ss.nodeMap[args.ServerInfo]
	//if not we have to add it to the map and to the list
	if ok != true {
		//put it in the list
		<- ss.nodeListM
		ss.nodeList = append(ss.nodeList, args.ServerInfo)
		//put it in the map w/ it's index in the list just cause whatever bro
		//map is just easy way to check for duplicates anyway
		ss.nodeMap[args.ServerInfo] = len(ss.nodeList)
	} 

	//check to see if all nodes have registered
	if len(ss.nodeList) == ss.numNodes {
		//if so we are ready
		reply.Ready = true
	} else {
		//if not we aren't ready
		reply.Ready = false
	}

	//send back the list of servers anyway
	reply.Servers = ss.nodeList

	//unlock everything
	ss.nodeListM <- 1
	ss.nodeMapM <- 1
	//NOTE: having these two mutexes may cause weird problems, might want to look into just having one mutex that is used for both the 
	//node list and the node map since they are baiscally the same thing anyway.

	return nil
}

func (ss *Storageserver) GetServers(args *storageproto.GetServersArgs, reply *storageproto.RegisterReply) error {
	//this is what libstore calls on the master to get a list of all the servers
	//if the lenght of the nodeList is the number of nodes then we return ready and the list of nodes
	//otherwise we return false for ready and the list of nodes we have so far
	<- ss.nodeListM

	//check to see if all nodes have registered
	if len(ss.nodeList) == ss.numNodes {
		//if so we are ready
		reply.Ready = true
	} else {
		//if not we aren't ready
		reply.Ready = false
	}

	//send back the list of servers anyway
	reply.Servers = ss.nodeList

	ss.nodeListM <- 1

	return nil
}

func (ss *Storageserver) revokeLeases(key string) bool {
	//revokes all leases for a given key

	//return true if/when all leases have been revoked properly (clients respond Status = OK)
	//or maybe don't return anything cause loop until actually revoked?

	//since we lock around all calls of this function we should not have to lock in the function itself (hopefully)
	leaseList := ss.leaseMap[key]

	for i := 0; i < len(leaseList); i++ {
		//connect to each client holding the lease
		cli, err := rpc.DialHTTP("tcp", leaseList[i])
		if err != nil {
			//fmt.Printf("Could not connect to server %s, returning nil\n", leaseList[i])
			return false
		}
		//revoke the lease
		args := &storageproto.RevokeLeaseArgs{}
		args.Key = key
		reply := &storageproto.RevokeLeaseReply{}

		for reply.Status != storageproto.OK {
			cli.Call("CacheRPC.RevokeLease", args, reply)
			time.Sleep(2*time.Second)
		}
	} 

	delete(ss.leaseMap, key)

	return true
}

// RPC-able interfaces, bridged via StorageRPC.
// These should do something! :-)

func (ss *Storageserver) Get(args *storageproto.GetArgs, reply *storageproto.GetReply) error {

	if args.WantLease == true {
		//grant lease
		//is there any reason to not grant it?
		reply.Lease.Granted = true
		reply.Lease.ValidSeconds = storageproto.LEASE_SECONDS
		<- ss.leaseMapM
		leaseList, exists := ss.leaseMap[args.Key]
		if exists == true {
			leaseList = append(leaseList, args.LeaseClient)
		} else {
			leaseList = []string{}
			leaseList = append(leaseList, args.LeaseClient)
		}
		ss.leaseMap[args.Key] = leaseList
		ss.leaseMapM <- 1
	} else {
		reply.Lease.Granted = false
		reply.Lease.ValidSeconds = 0
	}

	<- ss.valMapM
	val, ok := ss.valMap[args.Key]
	if ok != true {
		reply.Status = storageproto.EKEYNOTFOUND
		reply.Value = ""
	} else {
		reply.Status = storageproto.OK
		reply.Value = val
	}
	ss.valMapM <- 1

	return nil
}

func (ss *Storageserver) GetList(args *storageproto.GetArgs, reply *storageproto.GetListReply) error {

	if args.WantLease == true {
		//grant lease
		//is there any reason to not grant it?
		reply.Lease.Granted = true
		reply.Lease.ValidSeconds = storageproto.LEASE_SECONDS
		<- ss.leaseMapM
		leaseList, exists := ss.leaseMap[args.Key]
		if exists == true {
			leaseList = append(leaseList, args.LeaseClient)
		} else {
			leaseList = []string{}
			leaseList = append(leaseList, args.LeaseClient)
		}
		ss.leaseMap[args.Key] = leaseList
		ss.leaseMapM <- 1
	} else {
		reply.Lease.Granted = false
		reply.Lease.ValidSeconds = 0
	}

	<- ss.listMapM
	list, ok := ss.listMap[args.Key]
	if ok != true {
		reply.Status = storageproto.EITEMNOTFOUND
		reply.Value = []string{}
	} else {
		reply.Status = storageproto.OK
		reply.Value = list
	}
	ss.listMapM <- 1

	return nil
}

func (ss *Storageserver) Put(args *storageproto.PutArgs, reply *storageproto.PutReply) error {

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	if exists == true {
		ss.revokeLeases(args.Key)
	}
	ss.leaseMapM <- 1

	return nil
}

func (ss *Storageserver) AppendToList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	if exists == true {
		ss.revokeLeases(args.Key)
	}
	ss.leaseMapM <- 1

 	return nil
}

func (ss *Storageserver) RemoveFromList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	if exists == true {
		ss.revokeLeases(args.Key)
	}
	ss.leaseMapM <- 1

 	return nil
}
