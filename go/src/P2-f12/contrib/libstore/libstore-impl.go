// Sample implementation of libstore
// Assumes 1 storage server (and hence doesn't cache client RPC connections)
// Does not support caching
package libstore

import (
	"P2-f12/official/lsplog"
	"P2-f12/official/storageproto"
	"net/rpc"
	"strings"
	"fmt"
	"os"
)

type Libstore struct {
	cli *rpc.Client
	nodelist []storageproto.Node
	flags int
	myhostport string
}

// Doesn't implement retry
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
	args := storageproto.GetServersArgs{}
	var reply storageproto.RegisterReply
	err = cli.Call("StorageRPC.GetServers", args, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Ready {
		ls.nodelist = reply.Servers
	}
	if ls.nodelist == nil {
		return nil, lsplog.MakeErr("Storage system not ready")
	}

	// do NOT connect to other storage servers here
	// this should be done in a lazy fashion upon the first use
	// of the storage server, and then the rpc connection can be cached

	return ls, nil
}

func (ls *Libstore) getServer(key string) (*rpc.Client, error) {
	// Use beginning of key to group related keys together
	precolon := strings.Split(key, ":")[0]
	keyid := Storehash(precolon)
	// Assuming 1 storage server so just taking first node
	_ = keyid // use keyid to compile
	// Doesn't implement rpc connection caching since there is only 1 node
	// Don't forget to properly synchronize when caching rpc connections
	cli := ls.cli
	return cli, nil
}


func (ls *Libstore) iGet(key string) (string, error) {
	wantlease := false
	args := &storageproto.GetArgs{key, wantlease, ls.myhostport}
	var reply storageproto.GetReply
	cli, err := ls.getServer(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error in get server\n")
		return "", err
	}
	err = cli.Call("StorageRPC.Get", args, &reply)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RPC failed: %s\n", err)
		return "", err
	}
	if reply.Status != storageproto.OK {
		return "", lsplog.MakeErr("Get failed:  Storage error")
	}
	return reply.Value, nil
}

func (ls *Libstore) iPut(key, value string) error {
	args := &storageproto.PutArgs{key, value}
	var reply storageproto.PutReply
	cli, err := ls.getServer(key)
	if err != nil {
		return err
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
	wantlease := false
	args := &storageproto.GetArgs{key, wantlease, ls.myhostport}
	var reply storageproto.GetListReply
	cli, err := ls.getServer(key)
	if err != nil {
		return nil, err
	}
	err = cli.Call("StorageRPC.GetList", args, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Status != storageproto.OK {
		return nil, lsplog.MakeErr("GetList failed:  Storage error")
	}
	return reply.Value, nil
}

func (ls *Libstore) iRemoveFromList(key, removeitem string) error {
	args := &storageproto.PutArgs{key, removeitem}
	var reply storageproto.PutReply
	cli, err := ls.getServer(key)
	if err != nil {
		return err
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
	cli, err := ls.getServer(key)
	if err != nil {
		return err
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
