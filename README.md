# fmutex/mutex

Package mutext is designated to provide simple mutex locking
based on filesystem hard links functionality.
Given filesystem link function must fail, if target file already exists,
which is true for the Linux and MacOS platforms.

The module is designated to be used in distributed environment, using common
resource of the filesystem, for example for synchnonization during
initialization of K8s pods.

Related sample program `fmutex` can be used as a mutex utility for shell scripts.

## Getting started

To install run:

```console
go get github.com/bry00/fmutex
```

For usage example see the source of the sample `fmutex utility`: [main.go](main.go)

For a quick start, below is very simple usage sample:

```go
package main

import (
	"log"
	"os"
	"github.com/bry00/fmutex/mutex"
)

const MUTEX_ID = "sample-mutex"
const MUTEX_ROOT := "/tmp"

func main() {
	mx, err := mutex.NewMutex(MUTEX_ROOT, MUTEX_ID)
	if err != nil {
		log.Fatalf("cannot create the mutex: %v", err)
	}
	defer mx.Unlock()
	mx.Lock()
	// Do something that needs to be synced
	fmt.Println("DONE")
}
```

## License

The package is released under [the MIT license](LICENSE).
