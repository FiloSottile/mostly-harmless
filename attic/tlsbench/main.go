package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	tls "github.com/google/boringssl/ssl/test/runner"
	"github.com/pkg/profile"
	"gopkg.in/cheggaaa/pb.v1"
)

type Config struct {
	Targets []struct {
		Name    string
		Address string // host:port
	}
	SNI                string
	Version            string // tls12, tls13
	AllowDowngrade     bool
	InsecureSkipVerify bool
	Parallel           int // If 0, matches number of CPUs
	Timeout            string
	Repeats            int
	Profile            bool // If true, dump profiles
}

type job struct {
	Name    string
	Address string
	Timeout time.Duration

	h, sh, ih *histogram
	bar       *pb.ProgressBar
	tlsConfig *tls.Config
}

var outputMu sync.Mutex

func main() {
	c := &Config{}
	if err := json.NewDecoder(os.Stdin).Decode(c); err != nil {
		log.Fatal(err)
	}

	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		log.Fatal(err)
	}

	var version uint16
	if c.Version == "tls13" {
		version = tls.VersionTLS13
	} else if c.Version == "tls12" {
		version = tls.VersionTLS12
	} else {
		log.Fatal("Invalid Version field")
	}
	tlsConfig := &tls.Config{
		MaxVersion:         version,
		ServerName:         c.SNI,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
	if !c.AllowDowngrade {
		tlsConfig.MinVersion = version
	}

	h := newHistogram()
	sh := newHistogram()
	ih := newHistogram()
	bar := pb.New(len(c.Targets) * c.Repeats)
	bar.ShowTimeLeft = false
	bar.ManualUpdate = true
	bar.Start()

	jobChan := make(chan *job)

	workers := c.Parallel
	if workers == 0 {
		workers = runtime.GOMAXPROCS(-1)
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for j := range jobChan {
				runJob(j)
			}
			wg.Done()
		}()
	}

	fmt.Print("\n")

	if c.Profile {
		defer profile.Start().Stop()
	}

	for n := 0; n < c.Repeats; n++ {
		for _, t := range c.Targets {
			jobChan <- &job{
				Name:      t.Name,
				Address:   t.Address,
				Timeout:   timeout,
				tlsConfig: tlsConfig,
				h:         h,
				sh:        sh,
				ih:        ih,
				bar:       bar,
			}
		}
	}
	close(jobChan)
	wg.Wait()

	bar.Finish()
	fmt.Printf("\n\nVersion: %v, AllowDowngrade: %v, SNI: %v, Parallel: %v, Targets: %v, Repeats: %v\n",
		c.Version, c.AllowDowngrade, c.SNI, c.Parallel, len(c.Targets), c.Repeats)
	fmt.Print("\nHandshake time:\n")
	h.Print(true)
	fmt.Printf("\nFastest: %s - Slowest: %s\n\n", h.fastest, h.slowest)
	fmt.Print("\nTime to ServerHello:\n")
	sh.Print(true)
	fmt.Printf("\nFastest: %s - Slowest: %s\n\n", sh.fastest, sh.slowest)
	fmt.Print("\nTime in flight:\n")
	ih.Print(true)
	fmt.Printf("\nFastest: %s - Slowest: %s\n\n", ih.fastest, ih.slowest)
}
