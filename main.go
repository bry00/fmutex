package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/bry00/fmutex/mutex"
)

const (
	FlagRoot    = "root"
	EnvRoot     = "FMUTEX_ROOT"
	FlagId      = "id"
	FlagSilent  = "s"
	FlagPulse   = "pulse"
	FlagRefresh = "refresh"
	FlagLimit   = "limit"
	FlagTimeout = "timeout"
)

var cmn = struct { // Common flags
	Root   string
	Id     string
	Silent bool
}{
	Root:   ifEmptyStr(os.Getenv(EnvRoot), os.TempDir()),
	Silent: false,
}

var lck = struct { // Lock flags
	Pulse   time.Duration
	Refresh time.Duration
	Limit   time.Duration
	Timeout time.Duration
}{
	Pulse:   mutex.DefaultPulse,
	Refresh: mutex.DefaultRefresh,
	Limit:   mutex.DefaultDeadTimeout,
}

const (
	CmdLock    = "lock"
	CmdRelease = "release"
	CmdUnlock  = "unlock" // An alias to CmdRelease
	CmdTest    = "test"
)

var (
	cmdLock    *flag.FlagSet
	cmdRelease *flag.FlagSet
	cmdTest    *flag.FlagSet
	cmdAll     []*flag.FlagSet
	cmdNames   []string
)

func init() {
	log.SetFlags(0)
	log.SetPrefix(fmt.Sprintf("%s: ", getProg(os.Args)))

	flag.Usage = usage
	flag.StringVar(&cmn.Root, FlagRoot, cmn.Root, "root directory for mutex(es)")
	flag.StringVar(&cmn.Id, FlagId, cmn.Id, "mutex id")
	flag.BoolVar(&cmn.Silent, FlagSilent, cmn.Silent, "silent execution")

	cmdLock = flag.NewFlagSet(CmdLock, flag.ExitOnError)
	cmdLock.DurationVar(&lck.Pulse, FlagPulse, lck.Pulse, "determines frequency of locking attempts")
	cmdLock.DurationVar(&lck.Refresh, FlagRefresh, lck.Refresh, "determines frequency of saving current timestamp in a locking file")
	cmdLock.DurationVar(&lck.Limit, FlagLimit, lck.Limit, "determines how long takes to consider given mutex as \"dead\"")
	cmdLock.DurationVar(&lck.Timeout, FlagTimeout, lck.Timeout, "locking timeout (if > 0)")

	cmdRelease = flag.NewFlagSet(CmdRelease, flag.ExitOnError)
	cmdTest = flag.NewFlagSet(CmdTest, flag.ExitOnError)

	cmdAll, cmdNames = mkCommands(cmdLock, cmdRelease, cmdTest)

}

func main() {
	flag.Parse()

	if isEmptyStr(cmn.Id) {
		log.Fatalf("Flag -%s is required.", FlagId)
	}

	if flag.NArg() < 1 {
		log.Fatalf("Parameter error - expected command, one of: %s", strings.Join(cmdNames, ", "))
	}

	if cmn.Silent {
		log.SetOutput(ioutil.Discard)
	}
	switch flag.Arg(0) {
	case CmdLock:
		cmdLock.Parse(flag.Args()[1:])
		doLock()
		if !cmn.Silent {
			fmt.Println("LOCKED")
		}
	case CmdRelease, CmdUnlock:
		cmdRelease.Parse(flag.Args()[1:])
		doUnlock()
		if !cmn.Silent {
			fmt.Println("RELEASED")
		}
	case CmdTest:
		cmdTest.Parse(flag.Args()[1:])
		os.Exit(doTest())

	default:
		log.Fatalf("Fatal parameter error - unknown command \"%s\", valid commands are: %s", flag.Arg(0),
			strings.Join(cmdNames, ", "))
	}
}

func doTest() int {
	m := newMutex()
	lockPath := m.LockPath()
	if tm := m.When(); tm.IsZero() {
		log.Printf("Mutex \"%s\" (%s) is unlocked", m.Id(), lockPath)
		return 1
	} else {
		log.Printf("Mutex \"%s\" (%s) is locked: %s", m.Id(), lockPath, tm.Format(time.RFC3339))
	}
	return 0
}

func doLock() {
	m := newMutex()
	if err := m.TryLock(lck.Timeout); err != nil {
		log.Fatalf("Cannot lock mutex \"%s\": %v", m.Id(), err)
	}
}

func doUnlock() {
	m := newMutex()
	if err := m.TryUnlock(); err != nil {
		log.Fatalf("Cannot unlock mutex \"%s\": %v", m.Id(), err)
	}
}

func newMutex() *mutex.Mutex {
	result, err := mutex.NewMutexExt(cmn.Root, cmn.Id, lck.Pulse, lck.Refresh, lck.Limit)
	if err != nil {
		log.Fatalf("Cannot create mutex \"%s\": %v", cmn.Id, err)
	}
	return result
}

func ifEmptyStr(str string, defaultStr string) string {
	if isEmptyStr(str) {
		return defaultStr
	}
	return str
}

func isEmptyStr(str string) bool {
	return strings.TrimSpace(str) == ""
}
func getProg(args []string) string {
	base := path.Base(args[0])
	if i := strings.LastIndex(base, "."); i < 0 {
		return base
	} else {
		return base[0:i]
	}
}

func mkCommands(cmds ...*flag.FlagSet) ([]*flag.FlagSet, []string) {
	var result []string
	for _, c := range cmds {
		result = append(result, c.Name())
	}
	return cmds, result
}

func usage() {
	prog := getProg(os.Args)
	fmt.Fprintf(os.Stderr, "Program %s s designated to lock/unlock file-based mutexes.\n"+
		"Usage:\n"+
		"\t%s [options] {%s} [command-specific options]\n\n"+
		"options:\n",
		prog, prog, strings.Join(cmdNames, ", "))
	flag.PrintDefaults()

	for _, c := range cmdAll {
		var options = 0
		c.VisitAll(func(_ *flag.Flag) { options++ })
		if options > 0 {
			fmt.Fprintf(os.Stderr, "\n%s's options:\n", c.Name())
			c.PrintDefaults()
		}
	}
	fmt.Fprintln(os.Stderr)
}
