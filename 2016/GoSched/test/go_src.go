

// STARTPINIT OMIT
// go/src/runtime/proc.go

func schedinit() {
...
procs := int(ncpu)
if n := atoi(gogetenv("GOMAXPROCS")); n > 0 {
	if n > _MaxGomaxprocs {
		n = _MaxGomaxprocs
	}
	procs = n
}
if procresize(int32(procs)) != nil {
	throw("unknown runnable goroutine during bootstrap")
}

...
// STOPPINIT OMIT

// STARTWORKFLOW OMIT
                                +-------------------- sysmon ---------------//----+ 
                                |                                                 |
                                |                                                 |
               +---+      +---+-------+                   +--------+          +---+---+
go func() ---> | G | ---> | P | local | <=== balance ===> | global | <--//--- | P | M |
               +---+      +---+-------+                   +--------+          +---+---+
                            |                                 |                 |
                            |      +---+                      |                 |
                            +----> | M | <--- findrunnable ---+--- steal <--//--+
                                   +---+
                                     |
                                     |
              +--- execute <----- schedule
              |                      |
              |                      |
              +--> G.fn --> goexit --+

1. go creates a new goroutine
2. newly created goroutine being put into local or global queue
3. A M is being waken or created to execute goroutine
4. Schedule loop
5. Try its best to get a goroutine to execute
6. Clear, reenter schedule loop
// STOPWORKFLOW OMIT

// STARTINITM OMIT
// go/src/runtime/proc.go

// Set max M number to 10000
sched.maxmcount = 10000
...
// Initialize stack space
stackinit()
...
// Initialize current M
mcommoninit(_g_.m)
// STOPINITM OMIT

// STARTGOMAXPROCS OMIT
func GOMAXPROCS(n int) int {
	if n > _MaxGomaxprocs {
		n = _MaxGomaxprocs
	}
	lock(&sched.lock)
	ret := int(gomaxprocs)
	unlock(&sched.lock)
	if n <= 0 || n == ret {
		return ret
	}

	stopTheWorld("GOMAXPROCS")

	// newprocs will be processed by startTheWorld
	newprocs = int32(n)

	startTheWorld()
	return ret
}
// STOPGOMAXPROCS OMIT

// STARTNEWPROC1 OMIT
func newproc1(fn *funcval, argp *uint8, narg int32, nret int32, callerpc uintptr) *g {
	_g_ := getg() // GET current G

	...

	_p_ := _g_.m.p.ptr() // GET idle G from current P's queue
	newg := gfget(_p_)
	if newg == nil {
		newg = malg(_StackMin)
		casgstatus(newg, _Gidle, _Gdead)
		allgadd(newg) // publishes with a g->status of Gdead so GC scanner doesn't look at uninitialized stack.
	}
// STOPNEWPROC1 OMIT

// STARTGOROUTINEQ OMIT
type p struct {
	// Available G's (status == Gdead)
	gfree    *g
	gfreecnt int32
}
type schedt struct {
	// Global cache of dead G's.
	gflock mutex
	gfree  *g
	ngfree int32
}
// STOPGOROUTINEQ OMIT

// STARTSTEALGOROUTINE OMIT
// Get from gfree list.
// If local list is empty, grab a batch from global list.
func gfget(_p_ *p) *g {
retry:
	gp := _p_.gfree
	if gp == nil && sched.gfree != nil {
		lock(&sched.gflock)
		for _p_.gfreecnt < 32 && sched.gfree != nil {
			_p_.gfreecnt++
			gp = sched.gfree
			sched.gfree = gp.schedlink.ptr()
			sched.ngfree--
			gp.schedlink.set(_p_.gfree)
			_p_.gfree = gp
		}
		unlock(&sched.gflock)
		goto retry
	}
// STOPSTEALGOROUTINE OMIT

// STARTSTEALGOROUTINE2 OMIT
// Finds a runnable goroutine to execute.
// Tries to steal from other P's, get g from global queue, poll network.
func findrunnable() (gp *g, inheritTime bool) {
	...
	// random steal from other P's
	for i := 0; i < int(4*gomaxprocs); i++ {
		if sched.gcwaiting != 0 {
			goto top
		}
		_p_ := allp[fastrand1()%uint32(gomaxprocs)]
		var gp *g
		if _p_ == _g_.m.p.ptr() {
			gp, _ = runqget(_p_)
		} else {
			stealRunNextG := i > 2*int(gomaxprocs) // first look for ready queues with more than 1 g
			gp = runqsteal(_g_.m.p.ptr(), _p_, stealRunNextG)
		}
		if gp != nil {
			return gp, false
		}
	}
	...
// STOPSTEALGOROUTINE2 OMIT
