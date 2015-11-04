package pbservice

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongServer = "ErrWrongServer"
)

const (
	Put						= 	"Put"
	Append				=		"Append"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	// You'll have to add definitions here.
	Id  int64
	Op	string
	// Field names must start with capital letters,
	// otherwise RPC will break.
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
}

type GetReply struct {
	Err   Err
	Value string
}


// Your RPC definitions here.
type ForwardArgs struct {
	// storage map[string]	string
	Key string
	Value string
	Op string
	Id  int64
}

type ForwardReply struct {
	Err		Err
}

type DupArgs struct {
	Storage map[string] string
	Record  map[int64]	bool
}

type DupReply struct {
	Err	Err
}
