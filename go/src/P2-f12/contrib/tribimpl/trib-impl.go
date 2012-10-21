package tribimpl

import (
	"P2-f12/official/tribproto"
)

type Tribserver struct {
	//storagemaster
	//hostport
	//libstore
}

func NewTribserver(storagemaster string, myhostport string) *Tribserver {

	//pass storagemaster and hostport to new libstore with flags for debugging
	//make new libstore

	//store other shits

	//make tribserver

	//return the thing

	return &Tribserver{}
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

	//check errors for any calls to libstore

	//check if subscriptor exists
	//then reply = NOSUCHUSER
	//return nil

	//check if user you are subscribing to
	//then reply = NOSUCHTARGETUSER
	//return nil

	//so if both exist
	//then do append to List (user:subscription, target)
	//then reply = OK
	//return nil

	return nil
}

func (ts *Tribserver) RemoveSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {

	//check errors for any calls to libstore

	//check if subscriptor exists
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
	return nil
}

func (ts *Tribserver) PostTribble(args *tribproto.PostTribbleArgs, reply *tribproto.PostTribbleReply) error {
	return nil
}

func (ts *Tribserver) GetTribbles(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	return nil
}

func (ts *Tribserver) GetTribblesBySubscription(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	return nil
}
