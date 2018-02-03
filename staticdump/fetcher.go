package staticdump

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/evepraisal/go-evepraisal/bolt"
	"github.com/evepraisal/go-evepraisal/typedb"
	"github.com/sethgrid/pester"
)

// StaticFetcher continually fetches a new static dump and updates the type database with the current types
type StaticFetcher struct {
	dbPath   string
	callback func(typeDB typedb.TypeDB)
	client   *pester.Client

	stop chan bool
	wg   *sync.WaitGroup
}

// NewStaticFetcher returns a new static data fetcher
func NewStaticFetcher(client *pester.Client, dbPath string, callback func(typeDB typedb.TypeDB)) (*StaticFetcher, error) {
	fetcher := &StaticFetcher{
		dbPath:   dbPath,
		callback: callback,
		client:   client,

		stop: make(chan bool),
		wg:   &sync.WaitGroup{},
	}

	err := fetcher.RunOnce()
	if err != nil {
		return nil, err
	}

	fetcher.wg.Add(1)
	go func() {
		defer fetcher.wg.Done()
		for {

			select {
			case <-time.After(6 * time.Hour):
			case <-fetcher.stop:
				return
			}

			err := fetcher.RunOnce()
			if err != nil {
				log.Printf("WARNING: Fetcher failed to run: %s", err)
			}

			log.Println("FETCHER RAN")
		}
	}()

	return fetcher, nil
}

// RunOnce will fetch, parse and call the callback with a fresh type database
func (f *StaticFetcher) RunOnce() error {
	staticDumpURL, err := FindLastStaticDumpURL(f.client)
	if err != nil {
		return err
	}
	//
	staticDumpURLBase := filepath.Base(staticDumpURL)
	typedbPath := filepath.Join(f.dbPath, "types-"+strings.TrimSuffix(staticDumpURLBase, filepath.Ext(staticDumpURLBase)))
	if _, err = os.Stat(typedbPath); os.IsNotExist(err) {
		err = f.loadTypes(typedbPath, staticDumpURL)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	log.Println("Done loading types", staticDumpURLBase)

	typeDB, err := bolt.NewTypeDB(typedbPath, false)
	if err != nil {
		return err
	}
	log.Println("done making new typedb")

	f.callback(typeDB)
	return nil
}

// Close cleans up the worker
func (f *StaticFetcher) Close() error {
	close(f.stop)
	f.wg.Wait()
	return nil
}

func (f *StaticFetcher) loadTypes(staticCacheFile string, staticDumpURL string) error {

	// TODO: I need to find a reliable source for this information..... CCP????
	// typeVolumes, err := downloadTypeVolumes(f.client)
	// if err != nil {
	// 	return err
	// }

	// avoid re-downloading the entire static dump if we already have it
	cachepath := staticCacheFile + ".zip"
	if _, err := os.Stat(cachepath); os.IsNotExist(err) {
		log.Printf("Downloading static dump to %s", cachepath)
		err = downloadTypes(f.client, staticDumpURL, cachepath)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	types, err := loadtypes(cachepath)
	if err != nil {
		return err
	}

	typeDB, err := bolt.NewTypeDB(staticCacheFile, true)
	if err != nil {
		return err
	}
	finished := false
	defer func() {
		if finished == true {
			typeDB.Close()
		} else {
			log.Println("Deleting new typedb because it was stopped before finishing")
			err = typeDB.Delete()
			if err != nil {
				log.Printf("Error deleting old typedb: %s", err)
			}
		}
	}()

	for i, t := range types {
		if i%1000 == 0 {
			select {
			case <-f.stop:
				return nil
			default:
			}
		}

		var volume float64
		var ok bool
		volume, ok = volumeGroupOverrides[t.GroupID]
		if ok {
			t.PackagedVolume = volume
		}

		volume, ok = volumeMarketGroupOverrides[t.MarketGroupID]
		if ok {
			t.PackagedVolume = volume
		}

		volume, ok = volumeItemOverrides[t.ID]
		if ok {
			t.PackagedVolume = volume
		}

		err = typeDB.PutType(t)
		if err != nil {
			return err
		}
	}
	finished = true
	log.Println("Finished typedb fetch")
	return nil
}
