package tribimpl

import (
	"P2-f12/official/tribproto"
	"P2-f12/contrib/libstore"
	"time"
	"strconv"
	"fmt"
	"math"
	"encoding/json"
)

type Tribserver struct {
	storagemaster string
	hostport string
	lstore *libstore.Libstore
}

func NewTribserver(storagemaster string, myhostport string) *Tribserver {
	lstore, err := libstore.NewLibstore(storagemaster, myhostport, 0)

	if err != nil { return nil }
	return &Tribserver{storagemaster: storagemaster, hostport: myhostport, lstore: lstore}
}

func (ts *Tribserver) CreateUser(args *tribproto.CreateUserArgs, reply *tribproto.CreateUserReply) error {
	fmt.Println("Start create user...")
	// Set responses by modifying the reply structure, like:
	// reply.Status = tribproto.EEXISTS

	//check errors for any calls to libstore

	//call libstore.get on the user
	_, exists := ts.lstore.Get(args.Userid)
	//if error is found, then user does not exist yet
	if exists != nil {
		err1 := ts.lstore.Put(args.Userid, "0")
		if err1 != nil {
			fmt.Println("Put super fail.")
			return err1
		}
		fmt.Println("Created user " + args.Userid)
		//then set reply to OK
		reply.Status = tribproto.OK
		return nil
	}

	//else, the user already exists and we can't create 
	//set reply to EEXISTS
	reply.Status = tribproto.EEXISTS
	return nil	
	
}

func (ts *Tribserver) AddSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {
	fmt.Println("Start add subscriptions...")
	userId := args.Userid
	targetId := args.Targetuser

	//check errors for any calls to libstore
	_, userErr := ts.lstore.Get(userId)
	_, targetErr := ts.lstore.Get(targetId)

	//if userErr != nil, then user doesn't exist
	if userErr != nil { 
		reply.Status = tribproto.ENOSUCHUSER
		fmt.Println("No such user!")
		return nil 
	}
	if targetErr != nil { 
		reply.Status = tribproto.ENOSUCHTARGETUSER
		fmt.Println("No such target!")
		return nil 
	}
	if userErr == targetErr {
		fmt.Println("You can't subscribe to yourself!")
		reply.Status = tribproto.OK
		return nil
	}

	subs, subErr := ts.lstore.GetList(args.Userid + ":subscriptions")
	if subErr == nil { 
		for i:=0; i < len(subs); i++ {
			if subs[i] == targetId {
				reply.Status = tribproto.EEXISTS
				return nil
			}
		}
	}	

	//if both exist
	//then do append to List (user:subscriptions, target)
	//then reply = OK
	//return nil
	appendErr := ts.lstore.AppendToList(userId + ":subscriptions", targetId)
	if appendErr != nil { return appendErr }
	fmt.Println("Added " + targetId + " to user " + userId + " subscriptions!")

	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) RemoveSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {
	fmt.Println("Start remove subscriptions...")
	userId := args.Userid
	targetId := args.Targetuser

	if userId == targetId {
		fmt.Println("You can't remove yourself from your own subscriptions!")
		reply.Status = tribproto.OK
		return nil
	}

	//check errors for any calls to libstore
	_, userErr := ts.lstore.Get(userId)
	_, targetErr := ts.lstore.Get(targetId)

	if userErr != nil { 
		reply.Status = tribproto.ENOSUCHUSER
		return nil 
	}
	if targetErr != nil { 
		reply.Status = tribproto.ENOSUCHTARGETUSER
		return nil 
	}

	//if both exist
	//then remove from List (user:subscriptions, target)
	//then reply = OK
	//return nil
	removeErr := ts.lstore.RemoveFromList(userId + ":subscriptions", targetId)
	if removeErr != nil { 
		fmt.Println("Remove error!")
		reply.Status = tribproto.ENOSUCHTARGETUSER
		return nil 
	}
	fmt.Println("Removed " + targetId + " from user " + userId + " subscriptions!")

	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) GetSubscriptions(args *tribproto.GetSubscriptionsArgs, reply *tribproto.GetSubscriptionsReply) error {
	fmt.Println("Start get subscriptions...")
	
	//check errors for any calls to libstore


	//check if subscriber exists
	_, userErr := ts.lstore.Get(args.Userid)
		fmt.Println("Succeeded from libstore GET")

	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		return nil 
	}
	
	//otherwise...
	//then do getList (user:subscriptions)
	//err indicates that there is no list yet, which means there are no subscriptions
	subs, err1 := ts.lstore.GetList(args.Userid + ":subscriptions")

	if err1 != nil {
		reply.Userids = []string{}
		fmt.Println("User " + args.Userid + " is not subscribed to anyone!")
		return nil
	}

	//if there's no list, return an empty list
	if subs == nil {
		reply.Userids = []string{}
	} else {
		reply.Userids = subs
	}

	fmt.Println("User " + args.Userid + "'s subscriptions:\n***" )
	for i:=0; i<len(reply.Userids); i++ {
		fmt.Printf("%v\t", reply.Userids[i])
	}
	fmt.Println("")

	//then reply = OK, list
	reply.Status = tribproto.OK
	//return nil
	return nil
}

func (ts *Tribserver) PostTribble(args *tribproto.PostTribbleArgs, reply *tribproto.PostTribbleReply) error {
	fmt.Println("Attempting to post tribble...")
	now := time.Now().UnixNano()
	//Make into Tribble struct.
	tribble := tribproto.Tribble{Userid: args.Userid, Posted: time.Unix(0, now), Contents: args.Contents}	

	//check errors for any calls to libstore


	//check if user exists
	_, userErr := ts.lstore.Get(args.Userid)
	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		return nil 
	}

	//else
	//append to list([user:tribbles], tribbleID)
	//put tribble [user:tribbleID] = tribble
	//return nil
	tribbleJSON, marshalErr := json.Marshal(tribble)
	if marshalErr != nil { return marshalErr }

	//format timestamps in hex 
	tribbleId := strconv.FormatInt(now, 16)
	ts.lstore.Put(args.Userid, tribbleId) //mostly for checking for existance for other functions
	ts.lstore.Put(args.Userid + ":" + tribbleId, string(tribbleJSON)) 
	ts.lstore.AppendToList(args.Userid + ":timestamps", tribbleId)
	fmt.Println("User " + args.Userid + " posted " + args.Contents)
	reply.Status = tribproto.OK
	return nil
}	

func (ts *Tribserver) GetTribbles(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	fmt.Println("Attempting to get tribbles...")
	//check errors for any calls to libstore

	//check if user exists
	_, userErr := ts.lstore.Get(args.Userid)
	fmt.Println("Succeeded from libstore GET")

	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		reply.Tribbles = nil
		fmt.Println("No such user!")
		return nil
	}
	
	//else
	//getList(user:timestamps) => 
	timestamps, listErr := ts.lstore.GetList(args.Userid + ":timestamps")

	//if there's an error in retrieving a list, it means there are no tribbles
	if listErr != nil {
		reply.Status = tribproto.OK
		reply.Tribbles = []tribproto.Tribble{}
		fmt.Println("User " + args.Userid + " has no tribbles!")
		return nil
	}
  //for 100 tribbles (or up to 100 tribbles) at end of array (newest pushed to end) 
  numTribs := int(math.Min(100, float64(len(timestamps))))
  // get(tribble ID) and push onto tribbles array
  var tribbles []tribproto.Tribble

  var i int
  count := numTribs
  for i = len(timestamps) - 1; count > 0; i-- {
  	jtrib, err2 := ts.lstore.Get(args.Userid + ":" + timestamps[i])
 		
 		if err2 != nil {
 			fmt.Println("Fail on Get with " + timestamps[i])
 			return err2
 		}

 		var trib tribproto.Tribble
 		jtribBytes := []byte(jtrib)
 		jerr := json.Unmarshal(jtribBytes, &trib)

 		if jerr != nil {return jerr }

 		
 		tribbles = append(tribbles, trib)
 		count--
 	}

 	//reply = OK, tribbles array

	fmt.Println("User " + args.Userid + "'s tribbles:\n***" )
	for i:=0; i<len(tribbles); i++ {
		fmt.Printf("%v\t", tribbles[i])
	}
	fmt.Println("")

 	reply.Status = tribproto.OK
 	reply.Tribbles = tribbles
	return nil
}

func (ts *Tribserver) GetTribblesBySubscription(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	fmt.Println("Attempting to get tribbles by subscription")
	//check errors for any calls to libstore

	//check if user exists
	_, userErr := ts.lstore.Get(args.Userid)

	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		fmt.Println("There is no user.")
		return nil	
	}
	
	//Get subscriptions
	subs, suberr := ts.lstore.GetList(args.Userid + ":subscriptions")

	//err indicates there are no subscriptions
	if suberr != nil {
		reply.Status = tribproto.OK
		reply.Tribbles = []tribproto.Tribble{}	
		fmt.Println("User " + args.Userid + " is not subscribed to anyone!")
		return nil
	}

	//for all users 
	usrTimestamps := make(map[string]([]string))
	totalTribs := 0
	for i := 0; i < len(subs); i++ {
		//getTribbles for each user
		timestamps, listErr := ts.lstore.GetList(subs[i] + ":timestamps")

		//if not empty
		if listErr == nil && len(timestamps) > 0 {
			usrTimestamps[subs[i]] = timestamps
			totalTribs += len(timestamps)
		}

	}
	//Go through all tribble lists, and create new list of most recent 100 tribbles 
	numTribs := int(math.Min(100, float64(totalTribs)))
	replyTribs := []tribproto.Tribble{}
	for j := 0; j < numTribs; j++ {
		var id string
		latest := ""
		
		for userId, timestamps := range usrTimestamps {
			if len(timestamps) > 0 {
				curr := timestamps[len(timestamps) - 1] 
				nanoCurr, _ := strconv.ParseInt(curr, 16, 64)
				nanoLatest, _ := strconv.ParseInt(latest, 16, 64)
				if nanoCurr > nanoLatest {
					latest = curr
					id = userId
				}
			}			
			
		}
		
		//error makes no sense because we have these timestamps from a previous get
		jtrib, ohgawderr := ts.lstore.Get(id + ":" + latest)
		if ohgawderr != nil { 
			return ohgawderr 
		}

 		var trib tribproto.Tribble
 		jtribBytes := []byte(jtrib)
 		jerr := json.Unmarshal(jtribBytes, &trib)

 		if jerr != nil { return jerr }

		replyTribs = append(replyTribs, trib)

		//new list without latest Trib
		timestamps := usrTimestamps[id]
		newArray := []string{}
		for k:=0; k < len(timestamps) - 1; k++ {
			newArray = append(newArray, timestamps[k])	
		} 
		 	
		usrTimestamps[id] = newArray

	}

	fmt.Println("User " + args.Userid + "'s subscription tribbles:\n***" )
	for i:=0; i<len(replyTribs); i++ {
		fmt.Printf("%v\t", replyTribs[i])
	}
	fmt.Println("")

	reply.Status = tribproto.OK
	reply.Tribbles = replyTribs

	return nil
}
