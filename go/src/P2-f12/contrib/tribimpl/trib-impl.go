package tribimpl

import (
	"P2-f12/official/tribproto"
	"P2-f12/contrib/libstore"
	"time"
	"strconv"
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
	result, err0 := ts.lstore.Get(args.Userid)

	if err0 != nil {
		return err0
	}
	//if they don't
	_, errUser := strconv.Atoi(result)
	if errUser != nil {
		//then reply = NOSUCHUSER, nil
		reply.Status = tribproto.ENOSUCHUSER
		reply.Userids = nil
		//return nil
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
	now := time.Now()
	//Make into Tribble struct.
	tribble := tribproto.Tribble{Userid: args.Userid, Posted: time.Unix(0, now.UnixNano()), Contents: args.Contents}	

	//check errors for any calls to libstore


	//check if user exists
	userExists, userErr := ts.lstore.Get(args.Userid)
	if userErr != nil { return userErr }

	userConv, userConverr := strconv.Atoi(userExists)

	//if not
	//then reply = NOSUCHUSER
	//return nil
	if userConverr != nil {
		reply.Status = tribproto.ENOSUCHUSER
		return nil
	}

	//else
	//append to list([user:tribbles], tribbleID)
	//put tribble [user:tribbleID] = tribble
	//return nil
	tribbleJSON, marshalErr := json.Marshal(tribble)
	if marshalErr != nil { return marshalErr }

	tribbleId := strconv.Itoa(userConv + 1)
	ts.lstore.Put(args.Userid, tribbleId)
	ts.lstore.Put(args.Userid + ":" + tribbleId, string(tribbleJSON))
	ts.lstore.AppendToList(args.Userid + ":tribbles", tribbleId)
	reply.Status = tribproto.OK
	return nil
}	

func (ts *Tribserver) GetTribbles(args *tribproto.GetTribblesArgs, reply *tribproto.GetTribblesReply) error {
	
	//check errors for any calls to libstore

	//check if user exists
	result, err0 := ts.lstore.Get(args.Userid)

	if err0 != nil {
		reply.Status = tribproto.ENOSUCHUSER
		reply.Tribbles = nil
		return nil
	}
	_, errUser := strconv.Atoi(result)
	if errUser != nil {
		//then reply = NOSUCHUSER, nil
		reply.Status = tribproto.ENOSUCHUSER
		reply.Tribbles = nil
		//return nil
		return nil
	}

	//else
	//getList(user:tribbles) => 
	tribs, err1 := ts.lstore.GetList(args.Userid + ":tribbles")

	if err1 != nil {
		return err1
	}
  //for 100 tribbles (or up to 100 tribbles) at end of array (newest pushed to end) 
  numTribs := int(math.Min(100, float64(len(tribs))))
  // get(tribble ID) and push onto tribbles array
  var tribbles []tribproto.Tribble

  var i int
  count := numTribs
  for i = len(tribs) - 1; count > 0; i-- {
  		jtrib, err2 := ts.lstore.Get(args.Userid + ":" + tribs[i])
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
	result, err0 := ts.lstore.Get(args.Userid)

	if err0 != nil {
		reply.Status = tribproto.ENOSUCHUSER
		return nil
	}
	_, errUser := strconv.Atoi(result)
	if errUser != nil { return errUser }

	//else
	//getList(user:subscriptions)
	argsS := &tribproto.GetSubscriptionsArgs{Userid: args.Userid}
	var reS tribproto.GetSubscriptionsReply

	err1 := ts.GetSubscriptions(argsS, &reS)

	var subs []string
	if reS.Status != tribproto.OK || err1 != nil {
		reply.Status = reS.Status
		return err1
	}
	subs = reS.Userids
	//for all users 
	usrTribs := make(map[string]([]tribproto.Tribble))
	totalTribs := 0
	for i := 0; i < len(subs); i++ {
		args := &tribproto.GetTribblesArgs{Userid: subs[i]}
		var re tribproto.GetTribblesReply

		//getTribbles for each user
		err2 := ts.GetTribbles(args, &re)

		if err2 != nil {
			return err2
		}

		if re.Status == tribproto.OK {
			if len(re.Tribbles) > 0 {
				usrTribs[subs[i]] = re.Tribbles
				totalTribs += len(re.Tribbles)
			}
		}

	}
	//Go through all tribble lists, and create new list of most recent 100 tribbles 
	numTribs := int(math.Min(100, float64(totalTribs)))
	replyTribs := []tribproto.Tribble{}
	for j := 0; j < numTribs; j++ {
		var id string
		var latestTrib tribproto.Tribble
		
		for userId, tribbles := range usrTribs {
			var currTrib tribproto.Tribble
			if len(tribbles) > 0 { currTrib = tribbles[0] }			
			if currTrib.Posted.After(latestTrib.Posted) == true || id == "" || latestTrib.Posted.IsZero() == true {
				latestTrib = currTrib
				id = userId
			}
		}
		
		replyTribs = append(replyTribs, latestTrib)

		//new list without latestTrib
		tribbles := usrTribs[id]
		newArray := []tribproto.Tribble{}
		for k:=1; k < len(tribbles); k++ {
			newArray = append(newArray, tribbles[k])	
		} 
		
		usrTribs[id] = newArray

	}

	reply.Status = tribproto.OK
	reply.Tribbles = replyTribs

	return nil
}
