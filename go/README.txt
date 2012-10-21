These files serve as the official distribution of source code,
documentation, executables, and compiled libraries for Project 2.

The following files are included in the distribution

  bin:				Binaries
    linux_amd64:		Executables for Linux
	(Contents are the same as for darwin_amd64, below)

    darwin_amd64:		Executables for MacOS X
	libclient		A testing program that uses libstore to get/set values
	storageserver		Reference implementation of storage server
	tribclient		Testing program to post/view/etc. tweets
	tribserver		Reference implementation of tribble server

  src:
    P2-f12:			Sources for Project 2
      contrib:			Subdirectory for your code
	libstore		Skeleton code
	storageimpl		Skeleton code
	tribimpl		Skeleton code
      official:			Subdirectory for official libraries & code
        cacherpc		Interface definition for a caching client
	libclient		A testing program that uses libstore to get/set values
	lsplog			Convenience logging functions from P1
	storageproto		The RPC protocol definition for the storage server
	storagerpc		Interface definition for a storage server
	storageserver		A main() function that runs your storage impl
	tribclient		Testing program to post/view/etc. tweets
	tribproto		The RPC protocol definition for tribbling
	tribserver		A main() function that runs your tribble server impl

Running the storage server:

  SINGLE NODE:
     ./storageserver  [optionally:  -port=XXXX  to use a different port]

  MANY NODES:
    First, select one of your nodes to be the master:
      ./storageserver -N=XXX   where XXX specifies the total # of nodes 
                               in the system, including the master.

     Let's say you're running all of the nodes on the same host,
     with the master on port 9009.  You don't need to specify the
     port for the others - if they're slaves, they'll pick a master.

    And then run the other nodes with the -master set appropriately.
    Putting that all together, you might run:

       ./storageserver -port=9009 -N=3
       ./storageserver -master="localhost:9009"
       ./storageserver  -master="localhost:9009"

Testing the storage server from the command line:

   A simple way to start testing your storage server is to use
   the provided libclient.  For example, you might run:

      ./libclient -port=9009 p mykey helloworld
      ./libclient -port=9009 g mykey

We will only test your storage servers running on a single host, using
"localhost" to tell them how to talk to each other.
