// https://github.com/jsidrach/mnm-digest
package mnmdigest

// Dependencies
import (
	"bytes"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/urlfetch"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
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
	ServerURL   string `yaml:"server_url"`
	MeneameURL  string `yaml:"meneame_url"`
	RefreshRate uint   `yaml:"refresh_rate"`
	MaxStories  uint   `yaml:"max_stories"`
	Dir         string
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
	HTML       string `datastore:",noindex"`
	RSS        string `datastore:",noindex"`
	LastDigest time.Time
}

// Past stories, to guarantee the uniqueness of new stories
type Story struct {
	ID             string
	URL            string
	Title          string
	UpdatesToFlush int
	Karma          int `datastore:"-"`
}

// Stories type - implements the sort.Interface
type Stories []Story

// Access constants
const (
	CONFIG_FILE    string = "./config.yaml"
	FIXED_KEYS     string = "FixedKeys"
	DIGEST_KEY     string = "Digest"
	STORY_KIND     string = "Story"
	TEMPLATE_RSS   string = "template.rss"
	TEMPLATE_HTML  string = "template.html"
	TEMPLATE_INNER string = "template_inner.html"
)

//
// Functions
//

// Initialize global variables and handlers
func init() {
	// Global variables
	_, thisFile, _, _ := runtime.Caller(1)
	thisDir := path.Dir(thisFile)
	filename := path.Join(thisDir, CONFIG_FILE)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		panic(err)
	}
	config.Dir = thisDir
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
			w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
			fmt.Fprintf(w, s.RSS)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	return uint(math.Floor(time.Now().Sub(s.LastDigest).Hours()/24)) >= config.RefreshRate
}

// Refresh the digest, storing the pages into the datastore
func refreshDigest(c context.Context) {
	// Need mutex so we don't refresh the digest concurrently
	refresh.Lock()
	defer refresh.Unlock()
	// Check again since it could have been updated while locked
	if digestNeedsRefresh(c) {
		// Retrieve a *long enough* news list from menéame, sorted by karma
		stories, err := getNewStories(c)
		// External error, don't update the digest
		if err {
			log.Println("Error while getting new stories")
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
func getNewStories(c context.Context) ([]Story, bool) {
	var stories []Story
	var seconds int = (12 + 24*int(config.RefreshRate)) * 60 * 60
	var qStories = config.MeneameURL + "/rss?time=" + strconv.Itoa(seconds)
	// Fetch stories
	// Output format: RSS
	//   Stories enclosed between <item></item>
	//     ID             <link>ID</link>
	//     URL            <meneame:url>URL</meneame:url>
	//     Title          <title>Title</title>
	//     UpdatesToFlush int
	//     Karma          <meneame:karma>Karma</meneame:karma>
	client := urlfetch.Client(c)
	fStories, err := client.Get(qStories)
	if err != nil {
		log.Println(err)
		return stories, true
	}
	defer fStories.Body.Close()
	bStories, err := ioutil.ReadAll(fStories.Body)
	if err != nil {
		log.Println(err)
		return stories, true
	}
	sStories := string(bStories)
	buffStories := strings.Split(sStories, "<item>")[1:]
	for _, buffStory := range buffStories {
		id := getTagContent(buffStory, "link")
		url := getTagContent(buffStory, "meneame:url")
		title := getTagContent(buffStory, "title")
		karma, err := strconv.Atoi(getTagContent(buffStory, "meneame:karma"))
		if err != nil {
			continue
		}
		Story := Story{id, url, title, int(config.RefreshRate) + 2, karma}
		stories = append(stories, Story)
	}
	sort.Sort(Stories(stories))
	log.Println("# of stories after fetching: " + strconv.Itoa(len(stories)))
	return stories, len(stories) == 0
}

// Filter the new stories, keeping only the unique ones, and returning a maximum of MaxStories
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
	log.Println("# of stories remaining after filter: " + strconv.Itoa(len(topStories)))
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
	// Common inner part
	timeNow := time.Now()
	date := timeNow.Format(time.RFC822)
	shortDate := timeNow.Format("2006-01-02")
	tInner := template.New("template_inner.html")
	templateInner, err := tInner.ParseFiles(path.Join(config.Dir, TEMPLATE_INNER))
	if err != nil {
		panic(err)
	}
	var innerBuffer bytes.Buffer
	err = templateInner.Execute(&innerBuffer, stories)
	if err != nil {
		panic(err)
	}
	innerHTML := innerBuffer.String()

	// HTML
	dataHTML := struct {
		ShortDate string
		InnerHTML string
	}{
		shortDate,
		innerHTML,
	}
	tHTML := template.New("template.html")
	templateHTML, err := tHTML.ParseFiles(path.Join(config.Dir, TEMPLATE_HTML))
	if err != nil {
		panic(err)
	}
	var htmlBuffer bytes.Buffer
	err = templateHTML.Execute(&htmlBuffer, dataHTML)
	if err != nil {
		panic(err)
	}
	html := htmlBuffer.String()

	// RSS
	dataRSS := struct {
		ServerURL string
		Date      string
		ShortDate string
		InnerHTML string
	}{
		config.ServerURL,
		date,
		shortDate,
		innerHTML,
	}
	tRSS := template.New("template.rss")
	templateRSS, err := tRSS.ParseFiles(path.Join(config.Dir, TEMPLATE_RSS))
	if err != nil {
		panic(err)
	}
	var rssBuffer bytes.Buffer
	err = templateRSS.Execute(&rssBuffer, dataRSS)
	if err != nil {
		panic(err)
	}
	rss := rssBuffer.String()

	return html, rss
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

// Get the content of a tag
func getTagContent(buffer string, tag string) string {
	initTag := "<" + tag + ">"
	endTag := "</" + tag + ">"
	initIndex := strings.Index(buffer, initTag) + len(initTag)
	endIndex := strings.Index(buffer, endTag)
	return buffer[initIndex:endIndex]
}

// Stories implementation of sort.Interface
func (s Stories) Len() int           { return len(s) }
func (s Stories) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Stories) Less(i, j int) bool { return s[i].Karma < s[j].Karma }
