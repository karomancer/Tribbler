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
	"log"
	"hash/fnv"
)

type ClientLease struct {
	Client string
	Response chan int
}

type LeaseInfo struct {
	Mutex chan int
	ValidUntil int64
	ClientLeases *[]ClientLease
	ResponseLock chan int
	Responses *[]int
}

type Storageserver struct {
	nodeid uint32
	portnum int
	isMaster bool
	numNodes int
	nodeList []storageproto.Node
	nodeListM chan int
	nodeMap map[storageproto.Node]int
	nodeMapM chan int
	
	//key -> client
	//e.g. dga:7492ef010d -> host:port
	leaseMap map[string]*LeaseInfo //maps key to the clients who hold a lease on that key
	leaseMapM chan int

	blockOnRevoke map[string]chan int //key to a chan that blocks GET and GETLIST while revoking leases
	blockOnRevokeM chan int

	connMap map[string]*rpc.Client
	connMapM chan int

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
		time.Sleep(1*time.Second)
		<- ss.leaseMapM
		leases := ss.leaseMap //don't want to be accessing leaseMap in a loop while blocking other processes
		ss.leaseMapM <- 1
		for key, leaseInfo := range leases {
			now := time.Now().UnixNano()
			<- leaseInfo.Mutex
			validUntil := leaseInfo.ValidUntil
			leaseInfo.Mutex <- 1
			if now > validUntil {	
				fmt.Println("Lease for " + key + " expired.")
				go ss.ClearCaches(key)			
			}
		}
	}
}

func (ss *Storageserver) ClearCaches(key string) {
	//deleting lease
	<- ss.leaseMapM
	_, exists := ss.leaseMap[key]
	ss.leaseMapM <- 1

	if exists == false { return }
	
	<- ss.leaseMapM
	delete(ss.leaseMap, key)
	ss.leaseMapM <- 1
	fmt.Println("Lease for " + key + " removed from lease map")
}


func NewStorageserver(master string, numnodes int, portnum int, nodeid uint32) *Storageserver {

	//fmt.Println("called new storage server")

	ss := &Storageserver{}

	//if no nodeid is provided, choose one randomly
	if nodeid == 0 {
		reallySeedTheDamnRNG()
		ss.nodeid = rand.Uint32()
	} else {
		//otherwise just take the one they gave you
		ss.nodeid = nodeid
	}

	ss.portnum = portnum

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

	ss.leaseMap = make(map[string]*LeaseInfo)
	ss.leaseMapM = make(chan int, 1)
	ss.leaseMapM <- 1

	ss.listMap = make(map[string][]string)
	ss.listMapM = make(chan int, 1)
	ss.listMapM <- 1

	ss.connMap = make(map[string]*rpc.Client)
	ss.connMapM = make(chan int, 1)
	ss.connMapM <- 1

	ss.valMap = make(map[string]string)
	ss.valMapM = make(chan int, 1)
	ss.valMapM <- 1

	ss.blockOnRevoke = make(map[string]chan int)
	ss.blockOnRevokeM = make(chan int, 1)
	ss.blockOnRevokeM <- 1

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
		info := storageproto.Node{HostPort: "localhost:" + strconv.Itoa(portnum), NodeID: ss.nodeid}
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
		//fmt.Println("Aquired nodeList lock NewStorageserver")
		ss.nodeList = reply.Servers
		log.Println("Successfully joined storage node cluster.")
		slist := ""
		for i := 0; i < len(ss.nodeList); i++ {
			res := fmt.Sprintf("{localhost:%v %v}", ss.portnum, ss.nodeid)
			slist += res
			if i < len(ss.nodeList) - 1 {
				slist += " "	
			}
		}
		log.Printf("Server List: [%s]", slist)
		ss.nodeListM <- 1
		//fmt.Println("released nodeList lock NewStorageserver")

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
		//fmt.Println("aquired nodelist lock NewStorageserver")
		//append self to nodeList and put self in map
		me := storageproto.Node{HostPort: "localhost:" + strconv.Itoa(portnum), NodeID: ss.nodeid}
		ss.nodeList = append(ss.nodeList, me)//some shit here})
		ss.nodeListM <- 1
		//fmt.Println("released nodelist lock NewStorageserver")
		<- ss.nodeMapM
		//fmt.Println("aquired nodeMap lock NewStorageserver")
		ss.nodeMap[me] = 0
		ss.nodeMapM <- 1
		//fmt.Println("released nodeMap lock NewStorageserver")
	}

	ss.srpc = storagerpc.NewStorageRPC(ss)
	rpc.Register(ss.srpc)
	go ss.GarbageCollector()

	//fmt.Println("started new server")
	/*fmt.Println(storageproto.Node{HostPort: "localhost:" + strconv.Itoa(portnum), NodeID: ss.nodeid})
	fmt.Printf("master? %v\n", ss.isMaster)
	fmt.Printf("numnodes? %v\n", ss.numNodes)*/

	return ss
}

// Non-master servers to the master
func (ss *Storageserver) RegisterServer(args *storageproto.RegisterArgs, reply *storageproto.RegisterReply) error {
	//called on master by other servers
	//first check if that server is alreayd in our map

	//fmt.Println("called register server")

	<- ss.nodeMapM
	//fmt.Println("aquired nodeMap lock RegisterServer")
	_, ok := ss.nodeMap[args.ServerInfo]
	//if not we have to add it to the map and to the list
	if ok != true {
		//put it in the list
		<- ss.nodeListM
		//fmt.Println("aquired nodeList lock RegisterServer")
		ss.nodeList = append(ss.nodeList, args.ServerInfo)
		ss.nodeListM <- 1
		//fmt.Println("release nodeList lock RegisterServer")
		//put it in the map w/ it's index in the list just cause whatever bro
		//map is just easy way to check for duplicates anyway
		ss.nodeMap[args.ServerInfo] = len(ss.nodeList)
	} 

	//check to see if all nodes have registered
	<- ss.nodeListM
	//fmt.Println("aquired nodeList lock RegisterServer")
	if len(ss.nodeList) == ss.numNodes {
		//if so we are ready
		reply.Ready = true
		log.Println("Successfully joined storage node cluster.")
		slist := ""
		for i := 0; i < len(ss.nodeList); i++ {
			res := fmt.Sprintf("{localhost:%v %v}", ss.portnum, ss.nodeid)
			slist += res
			if i < len(ss.nodeList) - 1 {
				slist += " "	
			}
		}
		log.Printf("Server List: [%s]", slist)
	} else {
		//if not we aren't ready
		reply.Ready = false
	}

	//send back the list of servers anyway
	reply.Servers = ss.nodeList

	//unlock everything
	ss.nodeListM <- 1
	//fmt.Println("released nodeList lock RegisterServer")
	ss.nodeMapM <- 1
	//fmt.Println("released nodeMap lock RegisterServer")
	//NOTE: having these two mutexes may cause weird problems, might want to look into just having one mutex that is used for both the 
	//node list and the node map since they are baiscally the same thing anyway.

	//fmt.Println(reply.Servers)
	//fmt.Printf("ready? %v\n", reply.Ready)

	return nil
}

func (ss *Storageserver) GetServers(args *storageproto.GetServersArgs, reply *storageproto.RegisterReply) error {
	//this is what libstore calls on the master to get a list of all the servers
	//if the lenght of the nodeList is the number of nodes then we return ready and the list of nodes
	//otherwise we return false for ready and the list of nodes we have so far
	//fmt.Println("called get servers")
	<- ss.nodeListM
	//fmt.Println("aquried nodelist lock GetServers")
	//check to see if all nodes have registered
	if len(ss.nodeList) == ss.numNodes {
		//if so we are ready
		//fmt.Println("we are ready")
		reply.Ready = true
	} else {
		//if not we aren't ready
		//fmt.Println("we aren't ready")
		reply.Ready = false
	}

	//send back the list of servers anyway
	reply.Servers = ss.nodeList

	ss.nodeListM <- 1
	//fmt.Println("released nodelist lock GetServers")

	//fmt.Println(reply.Servers)
	//fmt.Printf("ready? %v\n", reply.Ready)

	return nil
}

func Storehash(key string) uint32 {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	return hasher.Sum32()
}

func (ss *Storageserver) checkServer(key string) bool {

	//fmt.Println("called checkServer")
	//fmt.Printf("key: %v\n", key)

	precolon := strings.Split(key, ":")[0]
	keyid := Storehash(precolon)

	//fmt.Printf("keyid: %v\n", keyid)
	//fmt.Printf("nodeid: %v\n", ss.nodeid)

	if keyid > ss.nodeid {
		//fmt.Println("keyid is greater than node id!")

		//but we might have wraparound!
		greaterThanAll := true
		for i := 0; i < len(ss.nodeList); i++ {
			if keyid < ss.nodeList[i].NodeID {
				greaterThanAll = false
			}
		}

		// if the key does need to be wrapped around, we need to make sure
		// it goes to the right server still, so we need to make sure our node
		// has the min node id, otherwise it's not the right one
		if greaterThanAll == true {
			lessThanAll := true
			for i := 0; i < len(ss.nodeList); i++ {
				if ss.nodeid > ss.nodeList[i].NodeID {
					lessThanAll = false
				}
			}

			//if it's not the least node id we have the wrong server
			if lessThanAll == false {
				return false
			} else {
				//otherwise we good
				return true
			}
		}

		// if the key doesn't need to be wrapped around, it's just in the wrong node
		return false
	}

	for i := 0; i < len(ss.nodeList); i++ {
		if keyid <= ss.nodeList[i].NodeID && ss.nodeList[i].NodeID < ss.nodeid {
			//fmt.Println("keyid is less than a lesser nodeId!")
			//fmt.Printf("keyid: %v, nodeid: %v\n", keyid, ss.nodeList[i].NodeID)
			return false
		}
	}

	//fmt.Println("key is on the right server")

	return true

}


func (ss *Storageserver) dialAndRPCRevoke(key string, li *LeaseInfo, cl ClientLease) {
	//connect to each client holding the lease

	
	//fmt.Printf("called dialAndRPCRevoke for key: %v\n", key)

	var cli *rpc.Client
	var exists bool
	var err error

	<- ss.connMapM 
	cli, exists = ss.connMap[cl.Client]
	ss.connMapM <- 1

	if exists == false {
		cli, err = rpc.DialHTTP("tcp", cl.Client)
		if err != nil {
			//fmt.Printf("Could not connect to server %s, returning nil\n", cl.Client)
			return 
		}
		<- ss.connMapM
		ss.connMap[cl.Client] = cli
		ss.connMapM <- 1
	} 
				
	<- ss.connMapM 
	cli, exists = ss.connMap[cl.Client]
	ss.connMapM <- 1

	//revoke the lease
	args := storageproto.RevokeLeaseArgs{Key: key}
	var reply storageproto.RevokeLeaseReply
	status := -1
	<- li.Mutex
	runUntil := li.ValidUntil
 	li.Mutex <- 1
 	now := time.Now().UnixNano()
	for status != storageproto.OK && now <= runUntil {
		err := cli.Call("CacheRPC.RevokeLease", &args, &reply)
		if err != nil {
			fmt.Println("Could not revoke lease.")
			break 
		}
		status = reply.Status
		now = time.Now().UnixNano()
	}
	if now > runUntil {
		fmt.Println("Lease expired before revokes complete.")
		return
	}
 
	<- li.ResponseLock
	*li.Responses = append(*li.Responses, 1)
	li.ResponseLock <- 1
}

func (ss *Storageserver) collectResponses(key string, numclients int, li *LeaseInfo) {
	fmt.Println("Collect got lock!")
	<- ss.blockOnRevoke[key]
	validUntil := li.ValidUntil
	for {
		<- li.Mutex
		responses := *li.Responses
		li.Mutex <- 1
		//If all revocations completed
		if len(responses) == numclients { break }

		//If it expired in the mean time
		now := time.Now().UnixNano()
		if now >= validUntil { 
			fmt.Println("Lease expired.")
			ss.ClearCaches(key)
			ss.blockOnRevoke[key] <- 1
			return 
		}
		time.Sleep(1*time.Second)
	}

	ss.ClearCaches(key)
 	
	fmt.Println("Revoked Lease.")
	ss.blockOnRevoke[key] <- 1
}

func (ss *Storageserver) revokeLeases(key string) {
	//revokes all leases for a given key

	//return true if/when all leases have been revoked properly (clients respond Status = OK)
	//or maybe don't return anything cause loop until actually revoked?

	<- ss.leaseMapM
	leaseInfo, _ := ss.leaseMap[key]
	ss.leaseMapM <- 1

	<- leaseInfo.Mutex  
	leaseList := *leaseInfo.ClientLeases
	leaseInfo.Mutex <- 1

	fmt.Println("Going to try and revoke lease...")
	go ss.collectResponses(key, len(leaseList), leaseInfo)

	for i := 0; i < len(leaseList); i++ {
		go ss.dialAndRPCRevoke(key, leaseInfo, leaseList[i])	
	}

}

// RPC-able interfaces, bridged via StorageRPC.
// These should do something! :-)

func (ss *Storageserver) Get(args *storageproto.GetArgs, reply *storageproto.GetReply) error {
	fmt.Println("called get for " + args.Key)

	//fmt.Println("called get")

	<- ss.blockOnRevokeM
	_, blockExists := ss.blockOnRevoke[args.Key]
	ss.blockOnRevokeM <- 1

	if blockExists == true {
		fmt.Println("Get got lock!")
		<- ss.blockOnRevoke[args.Key]
	}


	rightServer := ss.checkServer(args.Key)

	if rightServer == false {
		//fmt.Println("wrong server Get")
		reply.Status = storageproto.EWRONGSERVER
		if blockExists == true {
			ss.blockOnRevoke[args.Key] <- 1
		}
		return nil
	}

	<- ss.leaseMapM
	leaseInfo, exists := ss.leaseMap[args.Key]
	ss.leaseMapM <- 1

	
	if exists == true {
		list := *leaseInfo.ClientLeases
		for i:=0; i<len(list); i++ {
			if list[i].Client == args.LeaseClient {
				args.WantLease = false	
				break
			}
		}
	}

	if args.WantLease == true {		
		reply.Lease.Granted = true
		reply.Lease.ValidSeconds = storageproto.LEASE_SECONDS
		var keys []ClientLease

		if exists == true { 
			<- leaseInfo.Mutex
			*leaseInfo.ClientLeases = append(*leaseInfo.ClientLeases, ClientLease{Client: args.LeaseClient, Response: make(chan int, 1)})
			leaseInfo.Mutex <- 1
			//fmt.Printf("**GET added %v to %v lease list\n", args.LeaseClient, args.Key)
		} else {
			keys = []ClientLease{}
			keys = append(keys, ClientLease{Client: args.LeaseClient, Response: make(chan int, 1)})
			<- ss.leaseMapM 
			mutex := make(chan int, 1)
			mutex <- 1
			rlock := make(chan int, 1)
			rlock <- 1
			leaseExpiration := time.Now().Add((storageproto.LEASE_SECONDS+storageproto.LEASE_GUARD_SECONDS)*time.Second).UnixNano()
			ss.leaseMap[args.Key] = &LeaseInfo{Mutex: mutex, ValidUntil: leaseExpiration, ResponseLock: rlock, ClientLeases: &keys, Responses: &[]int{}}
			ss.leaseMapM <- 1

			//set up block revoke as well
			if blockExists == false {
				<- ss.blockOnRevokeM
				ss.blockOnRevoke[args.Key] = make(chan int, 1)
				ss.blockOnRevoke[args.Key] <- 1
				ss.blockOnRevokeM <- 1
			}

			fmt.Printf("**GET added Lease %v to %v\n", args.LeaseClient, args.Key)
			//fmt.Printf("**GET added Lease %v to %v\n", args.LeaseClient, args.Key)
		}

	} else {
		fmt.Println("Totally doesn't want a lease")
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

	//fmt.Printf("Get val for key %v: %v\n", args.Key, reply.Value)

	if blockExists == true {
		ss.blockOnRevoke[args.Key] <- 1
	}

	return nil
}

func (ss *Storageserver) GetList(args *storageproto.GetArgs, reply *storageproto.GetListReply) error {

	//fmt.Println("called getList")

	<- ss.blockOnRevokeM
	_, blockExists := ss.blockOnRevoke[args.Key]
	ss.blockOnRevokeM <- 1

	if blockExists == true {
		fmt.Println("GetList got lock!")
		<- ss.blockOnRevoke[args.Key]
	}
	
	//fmt.Printf("key: %v\n", args.Key)

	rightServer := ss.checkServer(args.Key)

	if rightServer == false {
		reply.Status = storageproto.EWRONGSERVER
		if blockExists == true {
			ss.blockOnRevoke[args.Key] <- 1
		}
		return nil
	}

	<- ss.leaseMapM
	leaseInfo, exists := ss.leaseMap[args.Key]
	ss.leaseMapM <- 1

	
		
	if exists == true {
		list := *leaseInfo.ClientLeases
		for i:=0; i<len(list); i++ {
			if list[i].Client == args.LeaseClient {
				fmt.Println("This guy doesn't want a lease")
				args.WantLease = false	
				break
			}
		}
	}


	if args.WantLease == true {
		//grant lease
		//is there any reason to not grant it?
		reply.Lease.Granted = true
		reply.Lease.ValidSeconds = storageproto.LEASE_SECONDS
		if exists == true { 
			<- leaseInfo.Mutex
			*leaseInfo.ClientLeases = append(*leaseInfo.ClientLeases, ClientLease{Client: args.LeaseClient, Response: make(chan int, 1)})
			leaseInfo.Mutex <- 1
			fmt.Printf("**GETLIST added %v to %v lease list\n", args.LeaseClient, args.Key)
		} else {
			keys := []ClientLease{}
			keys = append(keys, ClientLease{Client: args.LeaseClient, Response: make(chan int, 1)})
			<- ss.leaseMapM 
			mutex := make(chan int, 1)
			mutex <- 1
			rlock := make(chan int, 1)
			rlock <- 1
			leaseExpiration := time.Now().Add((storageproto.LEASE_SECONDS+storageproto.LEASE_GUARD_SECONDS)*time.Second).UnixNano()
			ss.leaseMap[args.Key] = &LeaseInfo{Mutex: mutex, ValidUntil: leaseExpiration, ResponseLock: rlock, ClientLeases: &keys, Responses: &[]int{}}
			ss.leaseMapM <- 1

			if blockExists == false {
				<- ss.blockOnRevokeM
				ss.blockOnRevoke[args.Key] = make(chan int, 1)
				ss.blockOnRevoke[args.Key] <- 1
				ss.blockOnRevokeM <- 1
			}

			// fmt.Printf("**GETLIST added %v to %v lease list\n", args.LeaseClient, args.Key)
		}

	} else {
		reply.Lease.Granted = false
		reply.Lease.ValidSeconds = 0
	}

	<- ss.listMapM
	list, ok := ss.listMap[args.Key]
	ss.listMapM <- 1

	if ok != true {
		reply.Status = storageproto.EITEMNOTFOUND
		reply.Value = []string{}
	} else {
		reply.Status = storageproto.OK
		reply.Value = list
	}

	if blockExists == true {
		ss.blockOnRevoke[args.Key] <- 1
	}

	return nil
}

func (ss *Storageserver) Put(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	fmt.Println("called put for " + args.Key)
	rightServer := ss.checkServer(args.Key)

	if rightServer == false {
		//fmt.Println("wrong server Put")
		reply.Status = storageproto.EWRONGSERVER
		return nil
	}

	<- ss.blockOnRevokeM
	_, blockExists := ss.blockOnRevoke[args.Key]
	ss.blockOnRevokeM <- 1

	if blockExists == true {
		fmt.Println("Put got lock!")
		<- ss.blockOnRevoke[args.Key]
	}

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	ss.leaseMapM <- 1

	if exists == true {	ss.revokeLeases(args.Key) } 		

	<- ss.valMapM
	ss.valMap[args.Key] = args.Value
	ss.valMapM <- 1


	if blockExists == true {
		ss.blockOnRevoke[args.Key] <- 1
	}

	reply.Status = storageproto.OK
	return nil
}

func (ss *Storageserver) AppendToList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	fmt.Println("APPEND TO LIST!")

	rightServer := ss.checkServer(args.Key)

	if rightServer == false {
		reply.Status = storageproto.EWRONGSERVER
		return nil
	}

	<- ss.blockOnRevokeM
	_, blockExists := ss.blockOnRevoke[args.Key]
	ss.blockOnRevokeM <- 1

	if blockExists == true {
		fmt.Println("Append got lock!")
		<- ss.blockOnRevoke[args.Key]
	}

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	ss.leaseMapM <- 1

	if exists == true {	ss.revokeLeases(args.Key) }


	//fmt.Println("Unlock lease map!")

	//fmt.Println("Lock list map!")
	<- ss.listMapM	
	list, ok := ss.listMap[args.Key]
	if ok == false { ss.listMap[args.Key] = []string{} }
	ss.listMapM <- 1 

	for i:=0; i < len(list); i++ {
		if list[i] == args.Value {
			reply.Status = storageproto.EITEMEXISTS
			return nil
		}
	}
	<- ss.listMapM
	ss.listMap[args.Key] = append(list, args.Value)
	ss.listMapM <- 1


	if blockExists == true {
		ss.blockOnRevoke[args.Key] <- 1
	}

	reply.Status = storageproto.OK
 	return nil
}

func (ss *Storageserver) RemoveFromList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {

	fmt.Println("REMOVE FROM LIST!")
	//fmt.Printf("key: %v\n", args.Key)

	rightServer := ss.checkServer(args.Key)

	if rightServer == false {
		reply.Status = storageproto.EWRONGSERVER
		return nil
	}

	<- ss.blockOnRevokeM
	_, blockExists := ss.blockOnRevoke[args.Key]
	ss.blockOnRevokeM <- 1

	if blockExists == true {
		fmt.Println("RemoveFromListgot lock!")
		<- ss.blockOnRevoke[args.Key]
	}

	//if we are changing something that people have leases on we have to invalidate all leases
	<- ss.leaseMapM
	_, exists := ss.leaseMap[args.Key]
	ss.leaseMapM <- 1

	if exists == true {	ss.revokeLeases(args.Key) }

	<- ss.listMapM
	list, ok := ss.listMap[args.Key]
	ss.listMapM <- 1

	if ok == false {
		reply.Status = storageproto.EKEYNOTFOUND
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
		return nil
	}

	<- ss.listMapM
	ss.listMap[args.Key] = newlist
	ss.listMapM <- 1

	if blockExists == true {
		ss.blockOnRevoke[args.Key] <- 1
	}

	reply.Status = storageproto.OK
 	return nil
}
