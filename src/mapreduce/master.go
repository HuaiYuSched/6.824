package mapreduce

import "container/list"
import "fmt"


type WorkerInfo struct {
	address string
	// You can add definitions here.
}


// Clean up all workers by sending a Shutdown RPC to each one of them Collect
// the number of jobs each work has performed.
func (mr *MapReduce) KillWorkers() *list.List {
	l := list.New()
	for _, w := range mr.Workers {
		DPrintf("DoWork: shutdown %s\n", w.address)
		args := &ShutdownArgs{}
		var reply ShutdownReply
		ok := call(w.address, "Worker.Shutdown", args, &reply)
		if ok == false {
			fmt.Printf("DoWork: RPC %s shutdown error\n", w.address)
		} else {
			l.PushBack(reply.Njobs)
		}
	}
	return l
}

func (mr *MapReduce) RunMaster() *list.List {
	// Your code here

	mapchan, reducechan := make(chan int, mr.nMap), make(chan int, mr.nReduce)

	//Send the job finished message to this channel
	sendjob := func(work string, jobnum int, job JobType) bool{

		rly := new(DoJobReply)
		arg := new(DoJobArgs)
		arg.Operation = job
		arg.JobNumber = jobnum
		arg.File = mr.file
		//someone told me why I need to exchange Reduce number with map nmuber
		switch job{
			case "Map":
				arg.NumOtherPhase = mr.nReduce
			case "Reduce":
				arg.NumOtherPhase = mr.nMap
		}
		//block and wait the rpc return
		//Synchronous Call
		result := call(work, "Worker.DoJob", arg, &rly)
		return result
	}

	doneworker := make(chan string)
	for i := 0; i < mr.nMap; i++ {
		go func(mapnum int){
				var result bool
				var worker string
				again:
				select {
					case worker = <- mr.registerChannel:
						result = sendjob(worker, mapnum, "Map")
					case worker = <- doneworker:
						result = sendjob(worker, mapnum, "Map")
				}
				if result {
					mapchan <- mapnum
					doneworker <- worker
					return
				}else {
		//			fmt.Println("Failed to send job\n\n\n\n")
						goto again;
				}
		}(i)
	}
	for i := 0; i < mr.nMap; i++ {
		//all the map done, this loop out.
		<- mapchan
	}

	for i := 0; i < mr.nReduce; i++ {
		go func(reducenum int) {
				var result bool
				var worker string
				again:
				select {
				case worker = <- mr.registerChannel:
					result = sendjob(worker, reducenum, "Reduce")
				case worker = <- doneworker:
					result = sendjob(worker, reducenum, "Reduce")
				}
				if result {
		//			fmt.Println("finish one reduce")
					reducechan <- reducenum
					doneworker <- worker
					return
				}else {
				///	fmt.Println("Failed to send job\n\n\n\n")
				goto again;
				}
		}(i)
	}
	for i := 0; i < mr.nReduce; i++ {
		//all the reduce done, this loop out.
		  <- reducechan
//		fmt.Println("reduce finished", num)
	}
	fmt.Println("Reduce done")
	return mr.KillWorkers()
}
