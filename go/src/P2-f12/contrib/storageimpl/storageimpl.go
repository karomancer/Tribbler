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
	"P2-f12/official/storagerpc"
	crand "crypto/rand"
	"math"
	"strconv"
	"math/big"
	"time"
	"net/rpc"
	"math/rand"
	"strings"
	"fmt"
)

type Storageserver struct {
	nodeid uint32
	isMaster bool
	numNodes int
	nodeList []storageproto.Node
	nodeListM chan int
	nodeMap map[storageproto.Node]int
	nodeMapM chan int
	
	//key -> client
	//e.g. dga:7492ef010d -> host:port
	leaseMap map[string][]string //maps key to the clients who hold a lease on that key
	leaseMapM chan int

	//client@key -> leaseTime
	//e.g. host:port@dga:7492ef010d -> #nanoseconds
	clientLeaseMap map[string]int64 //maps from client to timestamp when lease runs out
	clientLeaseMapM chan int

	listMap map[string][]string
	listMapM chan int


	valMap map[string]string
	valMapM chan int

	srpc *storagerpc.StorageRPC
}

func reallySeedTheDamnRNG() {
	randint, _ := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	rand.Seed( randint.Int64())
}

func (ss *Storageserver) GarbageCollector() {
	for {
		time.Sleep((storageproto.LEASE_SECONDS + storageproto.LEASE_GUARD_SECONDS)*time.Second)
		now := time.Now().UnixNano()
		<- ss.clientLeaseMapM
		leases := ss.clientLeaseMap //don't want to be accessing leaseMap in a loop while blocking other processes
		ss.clientLeaseMapM <- 1
		for key, timestamp := range leases {
			if now >= timestamp {	
				ss.ClearCaches(key) 
			}
		}
	}
}

func (ss *Storageserver) ClearCaches(clientKey string) {
	keystring := strings.Split(clientKey, "@")
	client := keystring[0]
	key := keystring[1]

	//delete client's lease from clientmap
	<- ss.clientLeaseMapM 
	delete(ss.clientLeaseMap, clientKey)
	ss.clientLeaseMapM <- 1

	//delete client from list of clients in leaseKey map
	<- ss.leaseMapM
	list := ss.leaseMap[key]
	if len(list) == 1 {
		delete(ss.leaseMap, key)
		ss.leaseMapM <- 1
		return
	}
	newlist := []string{}
	for i:=0; i<len(list); i++ {
		if list[i] != client {
			newlist = append(newlist, list[i])
		}
	}
	ss.leaseMap[key] = newlist
	ss.leaseMapM <- 1
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

	if numnodes != 0 {
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

	ss.clientLeaseMap = make(map[string]int64)
	ss.clientLeaseMapM = make(chan int, 1)
	ss.clientLeaseMapM <- 1

	ss.listMap = make(map[string][]string)
	ss.listMapM = make(chan int, 1)
	ss.listMapM <- 1

	ss.valMap = make(map[string]string)
	ss.valMapM = make(chan int, 1)
	ss.valMapM <- 1

	if ss.isMaster == false {
		ss.numNodes = 0
		//connect to the master server
		masterClient, err := rpc.DialHTTP("tcp", master) 
		for err != nil {
			//keep retrying until we can actually conenct
			//(Master may not have started yet)
			masterClient, err = rpc.DialHTTP("tcp", master)
			//fmt.Println("Trying to connect to master...")
			time.Sleep(time.Duration(3)*time.Second)	
		}

		//set up args for registering ourselves
		info := storageproto.Node{HostPort: strconv.Itoa(portnum), NodeID: ss.nodeid}
		args := storageproto.RegisterArgs{ServerInfo: info}
		reply := storageproto.RegisterReply{}

		for err != nil || reply.Ready != true {
			//call register on the master node with our info as the args. Kinda weird
			err = masterClient.Call("StorageRPC.Register", &args, &reply)
			//keep retrying until all things are registered
			//fmt.Println("Trying to register with master...")
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
		ss.numNodes = numnodes
		<- ss.nodeListM
		//append self to nodeList and put self in map
		me := storageproto.Node{HostPort: "localhost:" + strconv.Itoa(portnum), NodeID: ss.nodeid}
		ss.nodeList = append(ss.nodeList, me)//some shit here})
		ss.nodeListM <- 1
		<- ss.nodeMapM
		ss.nodeMap[me] = 0 //someshithere}] = 0
		ss.nodeMapM <- 1
	}

	ss.srpc = storagerpc.NewStorageRPC(ss)
	rpc.Register(ss.srpc)
	go ss.GarbageCollector()

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
		// fmt.Println("DIALING THE FUCK OUT OF HTTP! " + leaseList[i])
		if err != nil {
			fmt.Printf("Could not connect to server %s, returning nil\n", leaseList[i])
			return false
		}
		//revoke the lease
		args := &storageproto.RevokeLeaseArgs{Key: key}
		reply := &storageproto.RevokeLeaseReply{}

		for reply.Status != storageproto.OK {
			err = cli.Call("CacheRPC.RevokeLease", args, reply)
			if err != nil {
				fmt.Println("Could not revoke lease")
				return false
			}
			fmt.Println("Called CacheRPC revoke lease...")
			time.Sleep(2*time.Second)
		}
	} 

	delete(ss.leaseMap, key)

	return true
}

// RPC-able interfaces, bridged via StorageRPC.
// These should do something! :-)

func (ss *Storageserver) Get(args *storageproto.GetArgs, reply *storageproto.GetReply) error {

	fmt.Println("called get")
	fmt.Printf("key: %v\n", args.Key)

	fmt.Println("Want lease? " + strconv.FormatBool(args.WantLease))
	if args.WantLease == true {		
		<- ss.leaseMapM
		_, exists := ss.leaseMap[args.Key]	
		
		if exists == true {
			fmt.Println("Lease exists!")
			//for i:=0; i < len(list); i++ {
				//fmt.Println("Client: " + args.LeaseClient + "\t, Leases: " + list[i])
				//if list[i] == args.LeaseClient {
					fmt.Println("NO LEASE FOR YOU")
					reply.Status = storageproto.OK
					<- ss.valMapM
					reply.Value = ss.valMap[args.Key]
					ss.valMapM <- 1
					ss.leaseMapM <- 1
					return nil	
				//}
			//}
		}
		ss.leaseMapM <- 1
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

		<- ss.clientLeaseMapM
		leaseExpiration := time.Now().Add((storageproto.LEASE_SECONDS+storageproto.LEASE_GUARD_SECONDS)*time.Second).UnixNano()
		ss.clientLeaseMap[args.LeaseClient + "@" + args.Key] = leaseExpiration
		ss.clientLeaseMapM <- 1 
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

	fmt.Println("called getList")
	fmt.Printf("key: %v\n", args.Key)

	<- ss.leaseMapM
	list, exists := ss.leaseMap[args.Key]
	
	if exists == true {
		fmt.Println("have lease");
		for i:=0; i < len(list); i++ {
			if list[i] == args.LeaseClient {
				args.WantLease = false
				break		
			}
		}
	}
	ss.leaseMapM <- 1	

	if args.WantLease == true {
		fmt.Println("dont' have lease and want lease")
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

		<- ss.clientLeaseMapM
		leaseExpiration := time.Now().Add((storageproto.LEASE_SECONDS+storageproto.LEASE_GUARD_SECONDS)*time.Second).UnixNano()
		ss.clientLeaseMap[args.LeaseClient + "@" + args.Key] = leaseExpiration
		ss.clientLeaseMapM <- 1

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
		// fmt.Println("Revoke the things!")
		ss.revokeLeases(args.Key)
	}
	ss.leaseMapM <- 1

	<- ss.valMapM
	ss.valMap[args.Key] = args.Value
	ss.valMapM <- 1

	return nil
}

func (ss *Storageserver) AppendToList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	//fmt.Println("APPEND TO LIST!")

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	//fmt.Println("Lock lease map!")
	_, exists := ss.leaseMap[args.Key]
	if exists == true {
		//fmt.Println("SHIT EXISTS!")
		ss.revokeLeases(args.Key)
	}
	ss.leaseMapM <- 1
	//fmt.Println("Unlock lease map!")

	//fmt.Println("Lock list map!")
	<- ss.listMapM

	list, ok := ss.listMap[args.Key]
	if ok == false {
		ss.listMap[args.Key] = []string{}
	}

	for i:=0; i<len(list); i++ {
		if list[i] == args.Value {
			//fmt.Println("OH FUCK A DUPLICATE!")
			reply.Status = storageproto.EITEMEXISTS
			ss.listMapM <- 1
			return nil
		}
	}
	ss.listMap[args.Key] = append(ss.listMap[args.Key], args.Value)
	ss.listMapM <- 1 
	//fmt.Println("Unlock list map!")

	reply.Status = storageproto.OK
 	return nil
}

func (ss *Storageserver) RemoveFromList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	// //fmt.Println("REMOVE FROM LIST!")

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	if exists == true {
		ss.revokeLeases(args.Key)
	}
	ss.leaseMapM <- 1

	<- ss.listMapM
	list, ok := ss.listMap[args.Key]
	if ok == false {
		reply.Status = storageproto.EKEYNOTFOUND
		ss.listMapM <- 1
		return nil
	}

	j := -1
	newlist := []string{}
	for i:=0; i<len(list); i++ {
		if list[i] == args.Value {
			j = i
		} else {
			newlist = append(newlist, list[i])
		}
	}

	if j == -1 {
		reply.Status = storageproto.EITEMNOTFOUND
		ss.listMapM <- 1
		return nil
	}

	ss.listMap[args.Key] = newlist
	ss.listMapM <- 1

	reply.Status = storageproto.OK
 	return nil
}
