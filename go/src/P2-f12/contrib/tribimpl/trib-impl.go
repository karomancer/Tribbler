package tribimpl

import (
	"P2-f12/official/tribproto"
	"P2-f12/contrib/libstore"
	"time"
	"strconv"
	"fmt"
	// "strings"
	"math"
	// "errors"
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

	//pass storagemaster and hostport to new libstore with flags for debugging
	//make new libstore

	//store other shits

	//make tribserver

	//return the thing

	return &Tribserver{storagemaster: storagemaster, hostport: myhostport, lstore: lstore}
}

func (ts *Tribserver) CreateUser(args *tribproto.CreateUserArgs, reply *tribproto.CreateUserReply) error {
	// Set responses by modifying the reply structure, like:
	// reply.Status = tribproto.EEXISTS

	//check errors for any calls to libstore

	//call libstore.get on the user
	_, exists := ts.lstore.Get(args.Userid)
	//if error is found, then user does not exist yet
	if exists != nil {
		err1 := ts.lstore.Put(args.Userid, "0")
		if err1 != nil {
			return err1
		}
		//then set reply to OK
		reply.Status = tribproto.OK
		return nil
	}

	//else, the user already exists and we can't create 
	//set reply to EEXISTS
	reply.Status = tribproto.EEXISTS
	//return nil
	return nil

	
	
}

func (ts *Tribserver) AddSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {
	userId := args.Userid
	targetId := args.Targetuser

	//check errors for any calls to libstore
	_, userErr := ts.lstore.Get(userId)
	_, targetErr := ts.lstore.Get(targetId)

	//if userErr != nil, then user doesn't exist
	if userErr != nil { 
		reply.Status = tribproto.ENOSUCHUSER
		return nil 
	}
	if targetErr != nil { 
		reply.Status = tribproto.ENOSUCHTARGETUSER
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
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) RemoveSubscription(args *tribproto.SubscriptionArgs, reply *tribproto.SubscriptionReply) error {
	userId := args.Userid
	targetId := args.Targetuser

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
		reply.Status = tribproto.ENOSUCHTARGETUSER
		return nil 
	}
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) GetSubscriptions(args *tribproto.GetSubscriptionsArgs, reply *tribproto.GetSubscriptionsReply) error {
	
	//check errors for any calls to libstore


	//check if subscriber exists
	_, userErr := ts.lstore.Get(args.Userid)

	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		return nil 
	}
	
	//otherwise...
	//then do getList (user:subscriptions)
	subs, err1 := ts.lstore.GetList(args.Userid + ":subscriptions")

	if err1 != nil {
		return err1
	}
	//if there's no list, return an empty list
	if subs == nil {
		reply.Userids = []string{}
	} else {
		reply.Userids = subs
	}
	//then reply = OK, list
	reply.Status = tribproto.OK
	//return nil
	return nil
}

func (ts *Tribserver) PostTribble(args *tribproto.PostTribbleArgs, reply *tribproto.PostTribbleReply) error {
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
	fmt.Println("****POST! time_id: " + tribbleId)
	reply.Status = tribproto.OK
	return nil
}	

func (ts *Tribserver) GetTribbles(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	
	//check errors for any calls to libstore

	//check if user exists
	_, userErr := ts.lstore.Get(args.Userid)

	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		reply.Tribbles = nil
		return nil
	}
	
	//else
	//getList(user:timestamps) => 
	timestamps, listErr := ts.lstore.GetList(args.Userid + ":timestamps")

	//if there's an error in retrieving a list, it means there are no tribbles
	if listErr != nil {
		reply.Status = tribproto.OK
		reply.Tribbles = []tribproto.Tribble{}
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
 			return err2
 		}

 		var trib tribproto.Tribble
 		jtribBytes := []byte(jtrib)
 		jerr := json.Unmarshal(jtribBytes, &trib)

 		if jerr != nil {
 		 	return jerr
 		}

 		
 		tribbles = append(tribbles, trib)
 		count--
 	}

 	//reply = OK, tribbles array
 	reply.Status = tribproto.OK
 	reply.Tribbles = tribbles
 	//return nil
	return nil
}

func (ts *Tribserver) GetTribblesBySubscription(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	
	//check errors for any calls to libstore

	//check if user exists
	_, userErr := ts.lstore.Get(args.Userid)

	if userErr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		return nil
	}
	
	//else
	//getList(user:subscriptions)
	argsS := &tribproto.GetSubscriptionsArgs{Userid: args.Userid}
	var reS tribproto.GetSubscriptionsReply

	suberr := ts.GetSubscriptions(argsS, &reS)
	if suberr != nil {
		reply.Status = tribproto.OK
		reply.Tribbles = []tribproto.Tribble{}	
		return nil
	}

	var subs []string
	if reS.Status != tribproto.OK {
		reply.Status = reS.Status
		return nil
	}
	subs = reS.Userids
	//for all users 
	usrTimestamps := make(map[string]([]string))
	totalTribs := 0
	for i := 0; i < len(subs); i++ {
		args := &tribproto.GetTribblesArgs{Userid: subs[i]}

		//getTribbles for each user
		timestamps, listErr := ts.lstore.GetList(args.Userid + ":timestamps")

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
				curr := timestamps[0] 
				nanoCurr, _ := strconv.ParseInt(curr, 16, 64)
				nanoLatest, _ := strconv.ParseInt(latest, 16, 64)
				if nanoCurr > nanoLatest {
					latest = curr
					id = userId
				}
			}			
			
		}
		
		//error makes no sense because we have these timestamps from a previous get
		jtrib, ohgawderr := ts.lstore.Get(args.Userid + ":" + latest)
		if ohgawderr != nil { 
					fmt.Println("*****" + latest + "\t" + jtrib + "\t")

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
		for k:=1; k < len(timestamps); k++ {
			newArray = append(newArray, timestamps[k])	
		} 
		 	
		usrTimestamps[id] = newArray

	}

	reply.Status = tribproto.OK
	reply.Tribbles = replyTribs

	return nil
}
