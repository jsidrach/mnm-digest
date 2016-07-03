// Tests for the mnmdigest package
package mnmdigest

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/datastore"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Needed for datastore tests
var ctx context.Context

// Initialize test datastore
func init() {
	defer clearDatastore()
	// Do not output debug information
	config.Debug = false
	// Create new context, needed for datastore tests
	c, _, err := aetest.NewContext()
	if err != nil {
		panic(err)
	}
	ctx = c
}

// Should initialize global variables and handlers
func TestInit(t *testing.T) {
	defer clearDatastore()
	if config.Dir == "" {
		t.Errorf("The current directory should be set")
	}
}

// Should handle all requests
func TestHandleRequest(t *testing.T) {
	defer clearDatastore()
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatal("Instance should be able to be created")
	}
	defer inst.Close()
	// RSS request
	reqRSS, err := inst.NewRequest("GET", "/rss", nil)
	if err != nil {
		t.Fatal("/rss should be a valid request")
	}
	wRSS := httptest.NewRecorder()
	handleRequest(RSS)(wRSS, reqRSS)
	if wRSS.Code != 200 {
		t.Errorf("/rss should return HTTP Code 200")
	}
	// HTML request
	reqHTML, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("/ should be a valid request")
	}
	wHTML := httptest.NewRecorder()
	handleRequest(HTML)(wHTML, reqHTML)
	if wHTML.Code != 200 {
		t.Errorf("/ should return HTTP Code 200")
	}
}

// Should determine if the digest needs to be refreshed or not
func TestDigestNeedsRefresh(t *testing.T) {
	defer clearDatastore()
	config.RefreshRate = 1
	if !digestNeedsRefresh(ctx) {
		t.Errorf("Digest should need refresh initially")
	}
	refreshDigest(ctx)
	if digestNeedsRefresh(ctx) {
		t.Errorf("Digest should not need refresh after being refreshed")
	}
}

// Should refresh the digest, storing the pages into the datastore
func TestRefreshDigest(t *testing.T) {
	defer clearDatastore()
	config.RefreshRate = 1
	refreshDigest(ctx)
	if digestNeedsRefresh(ctx) {
		t.Errorf("Digest should not need refresh after being refreshed")
	}
	config.RefreshRate = 0
	if !digestNeedsRefresh(ctx) {
		t.Errorf("Digest should always need refresh if refresh rate is set to zero")
	}
}

// Should fetch the new stories from menéame
func TestGetNewStories(t *testing.T) {
	defer clearDatastore()
	config.RefreshRate = 1
	stories, err := getNewStories(ctx)
	storiesLen := len(stories)
	if err || storiesLen == 0 {
		t.Errorf("Number of fetches stories should be more than one")
	}
	for i := 1; i < storiesLen; i++ {
		if stories[i].Karma > stories[i-1].Karma {
			t.Errorf("Story #%v should not have higher karma than Story #%v", i, i-1)
		}
	}
}

// Should filter the new stories, keeping only the unique ones, and returning a maximum of MaxStories
func TestFilterNewStories(t *testing.T) {
	defer clearDatastore()
	config.MaxStories = 3
	story0 := Story{"ID0", "URL0", "TITLE0", 0, 4}
	story1 := Story{"ID1", "URL1", "TITLE1", 1, 3}
	story2 := Story{"ID2", "URL2", "TITLE2", 2, 2}
	story3 := Story{"ID3", "URL3", "TITLE3", 3, 1}
	story4 := Story{"ID4", "URL4", "TITLE4", 4, 0}
	storeStories(ctx, []Story{story0})
	stories := []Story{story0, story1, story2, story3, story4}
	testStories := filterNewStories(ctx, stories)
	testStoriesLen := len(testStories)
	if testStoriesLen != int(config.MaxStories) {
		t.Errorf("Number of filtered stories remaining should be %v not %v", config.MaxStories, testStoriesLen)
	}
	for i := range testStories {
		if !reflect.DeepEqual(testStories[i], stories[i+1]) {
			t.Errorf("Original Story #%v shold be equal to filtered Story #%v", i+1, i)
		}
	}
}

// Should update/delete the past stories
func TestUpdatePastStories(t *testing.T) {
	defer clearDatastore()
	story0 := Story{"ID0", "URL0", "TITLE0", 0, 0}
	story1 := Story{"ID1", "URL1", "TITLE1", 1, 0}
	story2 := Story{"ID2", "URL2", "TITLE2", 2, 0}
	story3 := Story{"ID3", "URL3", "TITLE3", -1, 0}
	story4 := Story{"ID4", "URL4", "TITLE4", 0, 0}
	stories := []Story{story0, story1, story2, story3, story4}
	storeStories(ctx, stories)
	updatePastStories(ctx)
	var testKeys []*datastore.Key
	for _, story := range stories {
		if story.UpdatesToFlush > 0 {
			testKeys = append(testKeys, datastore.NewKey(ctx, STORY_KIND, story.ID, 0, nil))
		}
	}
	testStories := make([]Story, len(testKeys))
	if err := datastore.GetMulti(ctx, testKeys, testStories); err != nil {
		panic(err)
	}
	testStoriesLen := len(testStories)
	remainingStoriesLen := 2
	if testStoriesLen != remainingStoriesLen {
		t.Errorf("Number of stored stories remaining should be %v not %v", remainingStoriesLen, testStoriesLen)
	}
}

// Should store the stories into the datastore
func TestStoreStories(t *testing.T) {
	defer clearDatastore()
	story0 := Story{"ID0", "URL0", "TITLE0", 0, 0}
	story1 := Story{"ID1", "URL1", "TITLE1", 1, 0}
	story2 := Story{"ID2", "URL2", "TITLE2", 2, 0}
	story3 := Story{"ID3", "URL3", "TITLE3", 3, 0}
	story4 := Story{"ID4", "URL4", "TITLE4", 4, 0}
	stories := []Story{story0, story1, story2, story3, story4}
	storeStories(ctx, stories)
	var testKeys []*datastore.Key
	for _, story := range stories {
		testKeys = append(testKeys, datastore.NewKey(ctx, STORY_KIND, story.ID, 0, nil))
	}
	testStories := make([]Story, len(testKeys))
	if err := datastore.GetMulti(ctx, testKeys, testStories); err != nil {
		panic(err)
	}
	testStoriesLen := len(testStories)
	storiesLen := len(stories)
	if testStoriesLen != storiesLen {
		t.Errorf("Number of stored stories should be %v not %v", storiesLen, testStoriesLen)
	}
	for i, testStory := range testStories {
		if !reflect.DeepEqual(stories[i], testStory) {
			t.Errorf("Story #%v should be equal before and after its storage", i)
		}
	}
}

// Should generate the pages
func TestGeneratePages(t *testing.T) {
	story0 := Story{"ID0", "URL0", "TITLE0", 0, 0}
	story1 := Story{"ID1", "URL1", "TITLE1", 0, 0}
	story2 := Story{"ID2", "URL2", "TITLE2", 0, 0}
	story3 := Story{"ID3", "URL3", "TITLE3", 0, 0}
	story4 := Story{"ID4", "URL4", "TITLE4", 0, 0}
	stories := Stories{story0, story1, story2, story3, story4}
	html, rss := generatePages(stories)
	for i, story := range stories {
		if !strings.Contains(html, story.ID) {
			t.Errorf("Generated HTML should contain Story #%v ID", i)
		}
		if !strings.Contains(rss, story.ID) {
			t.Errorf("Generated RSS should contain Story #%v ID", i)
		}
		if !strings.Contains(html, story.URL) {
			t.Errorf("Generated HTML should contain Story #%v URL", i)
		}
		if !strings.Contains(rss, story.URL) {
			t.Errorf("Generated RSS should contain Story #%v URL", i)
		}
		if !strings.Contains(html, story.Title) {
			t.Errorf("Generated HTML should contain Story #%v Title", i)
		}
		if !strings.Contains(rss, story.Title) {
			t.Errorf("Generated RSS should contain Story #%v Title", i)
		}
	}
}

// Should store and get the digest
func TestStoreAndGetDigest(t *testing.T) {
	defer clearDatastore()
	digest := Digest{"HTML", "RSS", time.Now()}
	storeDigest(ctx, &digest)
	var testDigest Digest
	if getDigest(ctx, &testDigest) != nil {
		t.Errorf("Digest should be stored be able to be retrieved after its storage")
	}
	// Cannot test property time - datastore changes its accuracy when storing it
	if digest.HTML != testDigest.HTML ||
		digest.RSS != testDigest.RSS {
		t.Errorf("Digest should be equal before and after its storage")
	}
}

// Should get the content of a tag
func TestGetTagContent(t *testing.T) {
	blob := "---<test>----<inner>content</inner>-</test>--"
	testOuterTagContent := getTagContent(blob, "test")
	outerTagContent := "----<inner>content</inner>-"
	if testOuterTagContent != outerTagContent {
		t.Errorf("Outer Tag Content should be " + outerTagContent + ", not " + testOuterTagContent)
	}
	testInnerTagContent := getTagContent(blob, "inner")
	innerTagContent := "content"
	if testInnerTagContent != innerTagContent {
		t.Errorf("Inner Tag Content should be " + innerTagContent + ", not " + testInnerTagContent)
	}
}

// Test the stories implementation of sort.Interface
func TestStoriesSortInterface(t *testing.T) {
	story0 := Story{"", "", "", 0, 0}
	story1 := Story{"", "", "", 0, 1}
	story2 := Story{"", "", "", 0, 2}
	story3 := Story{"", "", "", 0, 3}
	story4 := Story{"", "", "", 0, 5}
	stories := Stories{story0, story1, story2, story3, story4}
	storiesLen := stories.Len()
	if storiesLen != 5 {
		t.Errorf("Stories Length should be 5, not %v", storiesLen)
	}
	for i := 0; i < storiesLen; i++ {
		if stories.Less(i, i) {
			t.Errorf("Story #%v should not be below itself", i)
		}
		for j := i + 1; j < storiesLen; j++ {
			if !stories.Less(i, j) {
				t.Errorf("Story #%v should not be below Story #%v", j, i)
			}
			if stories.Less(j, i) {
				t.Errorf("Story #%v should not be below Story #%v", i, j)
			}
		}
	}
	stories.Swap(0, 1)
	if stories[0] != story1 || stories[1] != story0 {
		t.Errorf("Story #0 should have been swapped with Story #1")
	}
	if stories[2] != story2 {
		t.Errorf("Story #2 should be equal to itself")
	}
}

// Delete all the contents of the datastore
func clearDatastore() {
	keys, err := datastore.NewQuery("").KeysOnly().GetAll(ctx, nil)
	if err != nil {
		panic(err)
	}
	if err := datastore.DeleteMulti(ctx, keys); err != nil {
		panic(err)
	}
}
