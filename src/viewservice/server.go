package viewservice

import "net"
import "net/rpc"
import "log"
import "time"
import "sync"
import "fmt"
import "os"
import "sync/atomic"

type ViewServer struct {
	mu       sync.Mutex
	l        net.Listener
	dead     int32 // for testing
	rpccount int32 // for testing
	me       string


	// Your declarations here.
	View		View
	time map[string] time.Time
	//Current view's state is confirmed by primary or not
	// times map[sting] int
	acked 	bool
	//idlechan
	idlechan chan string
}

//
// server Ping RPC handler.
//
func (vs *ViewServer) Ping(args *PingArgs, reply *PingReply) error {

	// Your code here.
	vs.mu.Lock();
	vs.time[args.Me] = time.Now()
	if vs.View.Viewnum == 0 {
		vs.View.Primary = args.Me
		vs.View.Viewnum ++
		vs.acked = false
		// vs.time[args.Me] = time.Now()
		reply.View = vs.View
//		fmt.Println("get in");
//		fmt.Println(reply.View.Primary)
		vs.mu.Unlock();
		return nil
	}
//according to the name of server, determine the procedure.
	switch args.Me{
//The primary send Ping again, so make the acked as true
	case vs.View.Primary:
		//if the viewnum of ping is equal to the number of vs, then ack
		if vs.View.Viewnum == args.Viewnum {
			vs.acked = true
		//if the number of ping is zero, means the primary is failed
		}else if args.Viewnum == 0 {
			vs.View.Primary = vs.View.Backup
			vs.View.Backup = args.Me
			vs.View.Viewnum ++
			vs.acked = false
		}
		reply.View = vs.View
		vs.mu.Unlock();
		return nil
	case vs.View.Backup:
		reply.View = vs.View
		vs.mu.Unlock();
		return nil
//Not primary or backup -- idle server
	default:
		//If there is still no backup
		if vs.View.Backup == ""{
			if vs.acked == true{
				vs.View.Viewnum ++
				vs.View.Backup = args.Me
				vs.acked = false
				vs.time[args.Me] = time.Now()
			}
			reply.View=vs.View
			vs.mu.Unlock();
			return nil
		}else{		//idle server wait for machine failed
			vs.mu.Unlock();	//unlock to allowed others ping
			vs.idlechan <- args.Me	// send self to idlechan and wait receiving
		}
	}
	return nil
}

//
// server Get() RPC handler.
//
func (vs *ViewServer) Get(args *GetArgs, reply *GetReply) error {

	// Your code here.
	reply.View = vs.View
	return nil
}


//
// tick() is called once per PingInterval; it should notice
// if servers have died or recovered, and change the view
// accordingly.
//
func (vs *ViewServer) tick() {

	// Your code here.
	//check the time of primary and backup, if recently no activity, replace them
	// fmt.Println(time.Since(vs.time[vs.View.Primary]),DeadPings * PingInterval, vs.acked)

	if vs.View.Primary != ""{
		if time.Since(vs.time[vs.View.Primary]) > DeadPings * PingInterval && vs.acked == true{
			delete(vs.time, vs.View.Primary)
			vs.View.Primary = vs.View.Backup
			select {
			case  idleserver := <- vs.idlechan :
					vs.View.Backup = idleserver
			default :
					vs.View.Backup = ""
			}
			vs.View.Viewnum ++
			vs.acked = false
		}
	}
	if vs.View.Backup != "" {
		if time.Since(vs.time[vs.View.Backup]) > DeadPings * PingInterval {	//why do not need arcked?
			delete(vs.time, vs.View.Backup)
			select {
			case  idleserver := <- vs.idlechan :
					vs.View.Backup = idleserver
			default :
					vs.View.Backup = ""
			}
			vs.View.Viewnum ++
			vs.acked = false
		}
	}
}

//
// tell the server to shut itself down.
// for testing.
// please don't change these two functions.
//
func (vs *ViewServer) Kill() {
	atomic.StoreInt32(&vs.dead, 1)
	vs.l.Close()
}

//
// has this server been asked to shut down?
//
func (vs *ViewServer) isdead() bool {
	return atomic.LoadInt32(&vs.dead) != 0
}

// please don't change this function.
func (vs *ViewServer) GetRPCCount() int32 {
	return atomic.LoadInt32(&vs.rpccount)
}

func StartServer(me string) *ViewServer {
	vs := new(ViewServer)
	vs.me = me
	// Your vs.* initializations here.
	vs.View.Viewnum = 0;
	vs.time = make(map[string] time.Time)
	vs.View.Primary = ""
	vs.View.Backup = ""
	vs.idlechan = make(chan string)
	// vs.acked = false

	// tell net/rpc about our RPC server and handlers.
	rpcs := rpc.NewServer()
	rpcs.Register(vs)

	// prepare to receive connections from clients.
	// change "unix" to "tcp" to use over a network.
	os.Remove(vs.me) // only needed for "unix"
	l, e := net.Listen("unix", vs.me)
	if e != nil {
		log.Fatal("listen error: ", e)
	}
	vs.l = l

	// please don't change any of the following code,
	// or do anything to subvert it.

	// create a thread to accept RPC connections from clients.
	go func() {
		for vs.isdead() == false {
			conn, err := vs.l.Accept()
			if err == nil && vs.isdead() == false {
				atomic.AddInt32(&vs.rpccount, 1)
				go rpcs.ServeConn(conn)
			} else if err == nil {
				conn.Close()
			}
			if err != nil && vs.isdead() == false {
				fmt.Printf("ViewServer(%v) accept: %v\n", me, err.Error())
				vs.Kill()
			}
		}
	}()

	// create a thread to call tick() periodically.
	go func() {
		for vs.isdead() == false {
			vs.tick()
			time.Sleep(PingInterval)
		}
	}()

	return vs
}
