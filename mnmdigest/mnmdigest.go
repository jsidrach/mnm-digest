// https://github.com/jsidrach/mnm-digest
package mnmdigest

// Dependencies
import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/urlfetch"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
  "net/http"
	"strconv"
	"strings"
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
	MeneameAPI  string `yaml:"meneame_api"`
	RefreshRate uint   `yaml:"refresh_rate"`
	MaxStories  uint   `yaml:"max_stories"`
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
	ID             string
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
	// Global variables
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
	// Handlers
	http.HandleFunc("/", handleRequest(HTML))
	http.HandleFunc("/rss", handleRequest(RSS))
}

// Handle all requests
func handleRequest(t outputType) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Datastore context
		c := appengine.NewContext(r)
		// Refresh digest if needed
		if digestNeedsRefresh(c) {
			refreshDigest(c)
		}
		// Print the appropiate page
		var s Digest
		getDigest(c, &s)
		switch t {
		case RSS:
			fmt.Fprintf(w, s.RSS)
		default:
			fmt.Fprintf(w, s.HTML)
		}
	}
}

// Determine if the digest needs to be refreshed or not
func digestNeedsRefresh(c context.Context) bool {
	// Check if there is an existing cached version, if not, return true create a new one
	var s Digest
	err := getDigest(c, &s)
	if err != nil {
		return true
	}
	return uint(time.Now().Sub(s.LastDigest).Hours()) >= config.RefreshRate
}

// Refresh the digest, storing the pages into the datastore
func refreshDigest(c context.Context) {
	// Need mutex so we don't refresh the digest concurrently
	refresh.Lock()
	defer refresh.Unlock()
	// Check again since it could have been updated while locked
	if digestNeedsRefresh(c) {
		// Retrieve a *long enough* news list from menéame, sorted by karma
		stories, err := getNewStories()
		// External error, don't update the digest
		if err {
			return
		}
		// Filter out stories that have already appeared, and do not keep more than MaxStories
		topStories := filterNewStories(c, stories)
		// Update/delete past stories
		updatePastStories(c)
		// Store the unique list of new stories
		storeStories(c, topStories)
		// Generate the new pages
		html, rss := generatePages(topStories)
		// Store the new digest
		s := Digest{html, rss, time.Now()}
		storeDigest(c, &s)
	}
}

// Fetch the new stories from menéame
func getNewStories() ([]Story, bool) {
	var stories = make([]Story, 0, config.MaxStories)
	var days int = 1 + (int(config.RefreshRate) / 24)
	var rows int = (5 * int(config.MaxStories)) / 2
	var qStories = config.MeneameAPI + "/rank?days=" + strconv.Itoa(days) + "&rows=" + strconv.Itoa(rows)
	// Output format:
	//  Each story one line
	//  URL\tVotes\tNegatives\tKarma\n
	/*
	  fStories, err := http.Get(qStories)
		if err != nil {
	    return stories, true
		}
		defer fStories.Body.Close()
		bStories, err := ioutil.ReadAll(fStories.Body)
		if err != nil {
	    return stories, true
		}
		sStories := strings.Split(string(bStories), "\n")
		for _, sStory := range sStories {
			storyFields := strings.Split(sStory, "\t")
	    if len(storyFields) == 4 {
	  		URL := storyFields[0]
	  		story := Story{URL, URL, URL, 10}
	  		stories = append(stories, story)
	    }
		}
	*/
	// TODO
	// https://meneame.net/api/url?url=URL
	return stories, false
}

// Filters the new stories, keeping only the unique ones, and returning a maximum of MaxStories
func filterNewStories(c context.Context, stories []Story) []Story {
	var topStories = make([]Story, 0, config.MaxStories)
	for _, story := range stories {
		k := datastore.NewKey(c, STORY_KIND, story.ID, 0, nil)
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
	return topStories
}

// Update/delete the past stories
func updatePastStories(c context.Context) {
	// Remove the stories whose UpdateToFlush is zero
	// Decrease UpdatesToFlush for every story in past stories by one
	q := datastore.NewQuery(STORY_KIND)
	var pastStories []Story
	_, err := q.GetAll(c, &pastStories)
	if err != nil {
		panic(err)
	}
	for _, story := range pastStories {
		k := datastore.NewKey(c, STORY_KIND, story.ID, 0, nil)
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
}

// Store the stories into the datastore
func storeStories(c context.Context, stories []Story) {
	for _, story := range stories {
		k := datastore.NewKey(c, STORY_KIND, story.ID, 0, nil)
		_, err := datastore.Put(c, k, &story)
		if err != nil {
			panic(err)
		}
	}
}

// Generate the pages
func generatePages(stories []Story) (string, string) {
	/*
	   var tmp bytes.Buffer
	   t.Execute(&doc, template)
	   s := tmp.String()
	*/
	// TODO
	return "HTML", "RSS"
}

// Get the digest
func getDigest(c context.Context, s *Digest) error {
	k := datastore.NewKey(c, FIXED_KEYS, DIGEST_KEY, 0, nil)
	return datastore.Get(c, k, s)
}

// Store the digest
func storeDigest(c context.Context, s *Digest) {
	k := datastore.NewKey(c, FIXED_KEYS, DIGEST_KEY, 0, nil)
	_, err := datastore.Put(c, k, s)
	if err != nil {
		panic(err)
	}
}
