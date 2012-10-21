package tribimpl

import (
	"P2-f12/official/tribproto"
	"P2-f12/contrib/libstore"
)

type Tribserver struct {
	storagemaster string
	hostport string
	lstore *libstore.Libstore
}

func NewTribserver(storagemaster string, myhostport string) *Tribserver {
	lstore, _ := libstore.NewLibstore(storagemaster, myhostport, 0)
	return &Tribserver{storagemaster: storagemaster, hostport: myhostport, lstore: lstore}

	//pass storagemaster and hostport to new libstore with flags for debugging
	//make new libstore

	//store other shits

	//make tribserver

	//return the thing

}

func (ts *Tribserver) CreateUser(args *tribproto.CreateUserArgs, reply *tribproto.CreateUserReply) error {
	// Set responses by modifying the reply structure, like:
	// reply.Status = tribproto.EEXISTS

	//check errors for any calls to libstore

	//call libstore.get on the user
	//if it finds it
	//then it already exists and we can't create 
	//set reply to EEXISTS
	//return nil

	//if we don't find it
	//we create it with a PUT(user, 1)
	//then set reply to OK


	return nil
}

func (ts *Tribserver) AddSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {
	userId := args.Userid
	targetId := args.Targetuser

	//check errors for any calls to libstore
	userExists, userErr := ts.lstore.Get(userId)
	targetExists, targetErr := ts.lstore.Get(targetId)

	if userErr != nil { return userErr }
	if targetErr != nil { return targetErr }

	//check if subscriber exists
	//if not, then reply = NOSUCHUSER
	//return nil
	if userExists != "1" {
		reply.Status = tribproto.ENOSUCHUSER
		return nil
	}
	
	//if user subscribing to does not exist,
	//then reply = NOSUCHTARGETUSER
	//return nil
	if targetExists != "1" {
		reply.Status = tribproto.ENOSUCHTARGETUSER
		return nil
	}

	//if both exist
	//then do append to List (user:subscriptions, target)
	//then reply = OK
	//return nil
	ts.lstore.AppendToList(userId + ":subscriptions", targetId)
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) RemoveSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {

	//check errors for any calls to libstore

	//check if subscriber exists
	//then reply = NOSUCHUSER
	//return nil

	//check if user you are subscribing to
	//then reply = NOSUCHTARGETUSER
	//return nil

	//so if both exist
	//then do remove from List (user:subscription, target)
	//then reply = OK
	//return nil

	return nil
}

func (ts *Tribserver) GetSubscriptions(args *tribproto.GetSubscriptionsArgs, reply *tribproto.GetSubscriptionsReply) error {
	
	//check errors for any calls to libstore

	//check if subscriber exists
	//then reply = NOSUCHUSER, nil
	//return nil

	//check if user you are subscribing to
	//then reply = NOSUCHTARGETUSER, nil
	//return nil

	//so if both exist
	//then do get List (user:subscriptions, target)
	//if there's no list, return an empty list
	//then reply = OK, list
	//return nil


	return nil
}

func (ts *Tribserver) PostTribble(args *tribproto.PostTribbleArgs, reply *tribproto.PostTribbleReply) error {
		
	//check errors for any calls to libstore

	//check if user exists
	//if not
	//then reply = NOSUCHUSER
	//return nil

	//else
	//append to list([user:tribbles], tribbleID)
	//put tribble [user:tribbleID] = tribble
	//return nil

	return nil
}

func (ts *Tribserver) GetTribbles(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	
	//check errors for any calls to libstore

	//check if user exists
	//if not
	//then reply = NOSUCHUSER, nil
	//return nil

	//else
	//getList(user:tribbles) => 
  //for 100 tribbles (or up to 100 tribbles) at end of array (newest pushed to end) 
  //  get(tribble ID) and push onto tribbles array
  //reply = OK, tribbles array
  //return nil

	return nil
}

func (ts *Tribserver) GetTribblesBySubscription(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	
	//check errors for any calls to libstore

	//check if user exists
	//if not
	//then reply = NOSUCHUSER, nil
	//return nil

	//else
	//getList(user:subscriptions)
	//for all users 
	//	getTribbles for each user
	//Go through all tribble lists, and create new list of most recent 100 tribbles 

	return nil
}
