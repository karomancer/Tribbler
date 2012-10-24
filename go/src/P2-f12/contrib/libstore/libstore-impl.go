// Sample implementation of libstore
// Assumes 1 storage server (and hence doesn't connCache client RPC connections)
// Does not support caching
package libstore

import (
	"P2-f12/official/lsplog"
	"P2-f12/official/storageproto"
	"net/rpc"
	"strings"
	"fmt"
	"os"
	"time"
	"sort"
)

type Libstore struct {
	cli *rpc.Client
	nodelist []storageproto.Node
	connCache map [string]*rpc.Client
	cacheM chan int

	getCache map [string]string //map from user key to json string
	getM chan int

	getListCache map[string]([]string) //map from user key to array of json strings
	getListM chan int

	leaseMap map[string]storageproto.LeaseStruct
	leaseM chan int

	flags int
	myhostport string
}

type nodeL []storageproto.Node
func (nl nodeL) Len() int { return len(nl) }
func (nl nodeL) Swap(i, j int) { nl[i], nl[j] = nl[j], nl[i]}

type byID struct{ nodeL }
func (c byID) Less(i, j int) bool { return c.nodeL[i].NodeID < c.nodeL[j].NodeID }

func (ls *Libstore) leaseTimer(key string, seconds int) {
	timer := time.NewTimer(10*time.Second)
	<- ls.leaseM
	_, exists := ls.leaseMap[key]
	ls.leaseM <- 1
	if exists == false { return }


	select {
		case <- timer.C:
			<- ls.leaseM 
			delete(ls.leaseMap, key)
			ls.leaseM <- 1
	}
		
}


func iNewLibstore(server string, myhostport string, flags int) (*Libstore, error) {
	ls := &Libstore{}
	ls.flags = flags
	ls.myhostport = myhostport

	// Create RPC connection to storage server
	cli, err := rpc.DialHTTP("tcp", server)
	if err != nil {
		fmt.Printf("Could not connect to server %s, returning nil\n", server)
		return nil, err
	}
	ls.cli = cli

	// Get list of storage servers from master storage server
	// will retry five times after waiting for 1 second each try
	triesLeft := 5
	for triesLeft > 0 { 
		args := storageproto.GetServersArgs{}
		var reply storageproto.RegisterReply
		err = cli.Call("StorageRPC.GetServers", args, &reply)
		if err != nil {
			return nil, err
		}
		if reply.Ready {
			ls.nodelist = reply.Servers
			break
		}
		triesLeft --
		time.Sleep(1000000000)
	}

	if ls.nodelist == nil {
		return nil, lsplog.MakeErr("Storage system not ready")
	}

	sort.Sort(byID{ls.nodelist})

	//initialize the connCache
	ls.connCache = make(map[string]*rpc.Client)
	ls.cacheM = make(chan int, 1)
	ls.cacheM <- 1

	ls.getCache = make(map[string]string)
	ls.getM = make(chan int, 1)
	ls.getM <- 1

	ls.getListCache = make(map[string]([]string))
	ls.getListM = make(chan int, 1)
	ls.getListM <- 1

	ls.leaseMap = make(map[string]storageproto.LeaseStruct)

	ls.leaseM = make(chan int, 1)
	ls.leaseM <- 1

	// do NOT connect to other storage servers here
	// this should be done in a lazy fashion upon the first use
	// of the storage server, and then the rpc connection can be cached

	return ls, nil
}

func (ls *Libstore) getServer(key string) (*rpc.Client, error) {
	// Use beginning of key to group related keys together
	precolon := strings.Split(key, ":")[0]
	keyid := Storehash(precolon)
	//we gotta find out which dude we want to connect to
	fmt.Printf("Finding node for key: %v\n", keyid)
	var i int
	for i = 0; i < len(ls.nodelist); i++ {
		if ls.nodelist[i].NodeID > keyid {
			break
		}
	}
	nodeIndex := i
	//make the connection
	hostport := ls.nodelist[nodeIndex].HostPort
	cli, err := rpc.DialHTTP("tcp", hostport)
	if err != nil {
		fmt.Printf("Could not connect to server %s, returning nil\n", hostport)
		return nil, err
	}
	// rpc connection caching
	// Don't forget to properly synchronize when caching rpc connections
	<- ls.cacheM
	ls.connCache[key] = cli
	ls.cacheM <- 1

	return cli, nil
}


func (ls *Libstore) iGet(key string) (string, error) {
	//check if lease is still valid
	<- ls.leaseM 
	lease, found := ls.leaseMap[key]
	ls.leaseM <- 1

	if found == true && lease.Granted == true { 
		<- ls.getM
		thang := ls.getCache[key]
		ls.getM <- 1
		return thang, nil 
	} 

	wantlease := true
	args := &storageproto.GetArgs{key, wantlease, ls.myhostport}
	var reply storageproto.GetReply


	//check connCache for key
	<- ls.cacheM
	node, ok := ls.connCache[key]
	ls.cacheM <- 1
	//if it isn't found, we need to figure out which node to connect to

	cli := node
	var err error
	if ok != true {
		cli, err = ls.getServer(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in get server\n")
			return "", err
		}
	}

	err = cli.Call("StorageRPC.Get", args, &reply)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RPC failed: %s\n", err)
		return "", err
	}
	if reply.Status != storageproto.OK {
		return "", lsplog.MakeErr("Get failed:  Storage error")
	}
	
	<- ls.leaseM
	ls.leaseMap[key] = storageproto.LeaseStruct{Granted: true, ValidSeconds: storageproto.LEASE_SECONDS}
	ls.leaseM <- 1

	<- ls.getM
	ls.getCache[key] = reply.Value
	ls.getM <- 1

	go ls.leaseTimer(key, storageproto.LEASE_SECONDS)

	return reply.Value, nil
}

func (ls *Libstore) iPut(key, value string) error {
	args := &storageproto.PutArgs{key, value}
	var reply storageproto.PutReply

	//check connCache for key
	<- ls.cacheM
	node, ok := ls.connCache[key]
	ls.cacheM <- 1
	//if it isn't found, we need to figure out which node to connect to
	cli := node
	var err error
	if ok != true {
		cli, err = ls.getServer(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in get server\n")
			return err
		}
	}

	err = cli.Call("StorageRPC.Put", args, &reply)
	if err != nil {
		return err
	}
	if reply.Status != storageproto.OK {
		return lsplog.MakeErr("Put failed:  Storage error")
	}
	return nil
}

func (ls *Libstore) iGetList(key string) ([]string, error) {
	<- ls.leaseM
	_, found := ls.leaseMap[key]
	ls.leaseM <- 1

	if found == true { 
		<- ls.getM
		fmt.Println("Already in cache!")
		thang := ls.getListCache[key]
		ls.getM <- 1
		return thang, nil 
	} 

	wantlease := true
	args := &storageproto.GetArgs{key, wantlease, ls.myhostport}
	var reply storageproto.GetListReply

	//check connCache for key
	<- ls.cacheM
	node, ok := ls.connCache[key]
	ls.cacheM <- 1
	//if it isn't found, we need to figure out which node to connect to
	cli := node
	var err error
	if ok != true {
		cli, err = ls.getServer(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in get server\n")
			return nil, err
		}
	}

	err = cli.Call("StorageRPC.GetList", args, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Status != storageproto.OK {
		return nil, lsplog.MakeErr("GetList failed:  Storage error")
	}
	<- ls.leaseM
	ls.leaseMap[key] = storageproto.LeaseStruct{Granted: true, ValidSeconds: storageproto.LEASE_SECONDS}
	ls.leaseM <- 1

	<- ls.getListM
	ls.getListCache[key] = reply.Value
	ls.getListM <- 1

	go ls.leaseTimer(key, storageproto.LEASE_SECONDS)

	return reply.Value, nil
}


func (ls *Libstore) iRemoveFromList(key, removeitem string) error {
	args := &storageproto.PutArgs{key, removeitem}
	var reply storageproto.PutReply

	//check connCache for key
	<- ls.cacheM
	node, ok := ls.connCache[key]
	ls.cacheM <- 1
	//if it isn't found, we need to figure out which node to connect to
	cli := node
	var err error
	if ok != true {
		cli, err = ls.getServer(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in get server\n")
			return err
		}
	}

	err = cli.Call("StorageRPC.RemoveFromList", args, &reply)
	if err != nil {
		return err
	}
	if reply.Status != storageproto.OK {
		return lsplog.MakeErr("RemoveFromList failed:  Storage error")
	}
	return nil
}

func (ls *Libstore) iAppendToList(key, newitem string) error {
	args := &storageproto.PutArgs{key, newitem}
	var reply storageproto.PutReply

	//check connCache for key
	<- ls.cacheM
	node, ok := ls.connCache[key]
	ls.cacheM <- 1
	//if it isn't found, we need to figure out which node to connect to
	cli := node
	var err error
	if ok != true {
		cli, err = ls.getServer(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in get server\n")
			return err
		}
	}

	err = cli.Call("StorageRPC.AppendToList", args, &reply)
	if err != nil {
		return err
	}
	if reply.Status != storageproto.OK {
		return lsplog.MakeErr("AppendToList failed:  Storage error")
	}
	return nil
}
