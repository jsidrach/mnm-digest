// https://github.com/jsidrach/mnm-digest
package mnmdigest

// Dependencies
import (
	"fmt"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

//
// Global variables
//

// Configuration - Read-only after init
var config ConfigType

// Mutex for digest refresh
var refresh = &sync.Mutex{}

//
// Types
//

// Configuration
type ConfigType struct {
	MeneameURL  string `yaml:"meneame_url"`
	RefreshRate uint   `yaml:"refresh_rate"`
	MaxStories  uint   `yaml:"max_articles"`
}

// Type of output
type outputType uint

// The list of of all possible "enum" values
const (
	HTML outputType = iota
	RSS
)

// Datastore

// Stored pages and time they were generated
type Digest struct {
	HTML       string
	RSS        string
	LastDigest time.Time
}

// Past stories, to guarantee the uniqueness of new stories
type Story struct {
	URL            string
	Title          string
	UpdatesToFlush int
}

// Access constants
const (
	CONFIG_FILE string = "./config.yaml"
	FIXED_KEYS  string = "FixedKeys"
	DIGEST_KEY  string = "Digest"
	STORY_KIND  string = "Story"
)

//
// Functions
//

// Initialize global variables and handlers
func init() {
	// Initialize global variables
	filename, err := filepath.Abs("./config.yaml")
	if err != nil {
		panic(err)
	}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		panic(err)
	}

	// Initialize handlers
	http.HandleFunc("/", handleRequest(HTML))
	http.HandleFunc("/rss", handleRequest(RSS))
}

// Handle all requests
func handleRequest(t outputType) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Refresh digest if needed
		if digestNeedsRefresh(r) {
			refreshDigest(r)
		}

		// Print the appropiate page
		var s Digest
		getDigest(r, &s)
		switch t {
		case RSS:
			fmt.Fprintf(w, s.RSS)
		default:
			fmt.Fprintf(w, s.HTML)
		}
	}
}

// Determines if the digest needs to be refreshed or not
func digestNeedsRefresh(r *http.Request) bool {
	// Check if there is an existing cached version, if not, return true create a new one
	var s Digest
	err := getDigest(r, &s)
	if err != nil {
		return true
	}
	return uint(time.Now().Sub(s.LastDigest).Hours()) >= config.RefreshRate
}

// Refresh the digest, storing the pages into the datastore
func refreshDigest(r *http.Request) {
	// Need mutex so we don't refresh the digest concurrently
	refresh.Lock()

	// Check again since it could have been updated while locked
	if digestNeedsRefresh(r) {
		// Datastore context
		c := appengine.NewContext(r)

		// Retrieve a *long enough* news list from menéame, sorted by karma
		// TODO
    // https://github.com/crodas/Meneame.net/blob/master/www/api/rank.php
    // https://meneame.net/api/rank?rows=5&days=5
		var stories = make([]Story, 10)

		// Filter out stories that have already appeared, and do not keep more than MaxStories
		var topStories = make([]Story, 0, config.MaxStories)
		for _, story := range stories {
			k := datastore.NewKey(c, STORY_KIND, story.URL, 0, nil)
			// If story does not exists yet, add it to the unique stories
			err := datastore.Get(c, k, &story)
			if err != nil {
				topStories = append(topStories, story)
			}
			// Stop if we have enough stories
			if len(topStories) == int(config.MaxStories) {
				break
			}
		}

		// Decrease UpdatesToFlush for every story in past stories by one, removing the stories whose counter is zero
		q := datastore.NewQuery(STORY_KIND)
		var pastStories []Story
		_, err := q.GetAll(c, &pastStories)
		if err != nil {
			panic(err)
		}
		for _, story := range pastStories {
			k := datastore.NewKey(c, STORY_KIND, story.URL, 0, nil)
			if story.UpdatesToFlush == 0 {
				datastore.Delete(c, k)
			} else {
				story.UpdatesToFlush -= 1
				_, err := datastore.Put(c, k, &story)
				if err != nil {
					panic(err)
				}
			}
		}

		// Store the unique list of new stories
		for _, story := range topStories {
			k := datastore.NewKey(c, STORY_KIND, story.URL, 0, nil)
			_, err := datastore.Put(c, k, &story)
			if err != nil {
				panic(err)
			}
		}

		// Generate page_html and page_rss using the trimmed list of new stories
		// TODO

		// Store the new digest
		s := Digest{"HTML", "RSS", time.Now()}
		storeDigest(r, &s)
	}

	refresh.Unlock()
}

// Get the digest
func getDigest(r *http.Request, s *Digest) error {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, FIXED_KEYS, DIGEST_KEY, 0, nil)
	return datastore.Get(c, k, s)
}

// Store the digest
func storeDigest(r *http.Request, s *Digest) {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, FIXED_KEYS, DIGEST_KEY, 0, nil)
	_, err := datastore.Put(c, k, s)
	if err != nil {
		panic(err)
	}
}
