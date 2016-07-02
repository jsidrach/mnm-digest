// Tests for the mnmdigest package
package mnmdigest

import (
	"strconv"
	"testing"
)

// Should initialize global variables and handlers
func TestInit(t *testing.T) {
	// TODO
}

// Should handle all requests
func TestHandleRequest(t *testing.T) {
	// TODO
}

// Should determine if the digest needs to be refreshed or not
func TestDigestNeedsRefresh(t *testing.T) {
	// TODO
}

// Should refresh the digest, storing the pages into the datastore
func TestRefreshDigest(t *testing.T) {
	// TODO
}

// Should fetch the new stories from menéame
func TestGetNewStories(t *testing.T) {
	// TODO
}

// Should filter the new stories, keeping only the unique ones, and returning a maximum of MaxStories
func TestFilterNewStories(t *testing.T) {
	// TODO
}

// Should update/delete the past stories
func TestUpdatePastStories(t *testing.T) {
	// TODO
}

// Should store the stories into the datastore
func TestStoreStories(t *testing.T) {
	// TODO
}

// Should generate the pages
func TestGeneratePages(t *testing.T) {
	// TODO
}

// Should get the digest
func TestGetDigest(t *testing.T) {
	// TODO
}

// Should store the digest
func TestStoreDigest(t *testing.T) {
	// TODO
}

// Should get the content of a tag
func TestGetTagContent(t *testing.T) {
	blob := "---<test>----<inner>content</inner>-</test>--"
	testOuterTagContent := getTagContent(blob, "test")
	outerTagContent := "----<inner>content</inner>-"
	if testOuterTagContent != outerTagContent {
		t.Error("Outer Tag Content should be " + outerTagContent + ", not " + testOuterTagContent)
	}
	testInnerTagContent := getTagContent(blob, "inner")
	innerTagContent := "content"
	if testInnerTagContent != innerTagContent {
		t.Error("Inner Tag Content should be " + innerTagContent + ", not " + testInnerTagContent)
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
		t.Error("Stories Length should be 5, not " + strconv.Itoa(storiesLen))
	}
	for i := 0; i < 5; i++ {
		if stories.Less(i, i) {
			t.Error("Story #" + strconv.Itoa(i) + " should not be below Story #" + strconv.Itoa(i))
		}
		for j := i + 1; j < 5; j++ {
			if !stories.Less(i, j) {
				t.Error("Story #" + strconv.Itoa(j) + " should not be below Story #" + strconv.Itoa(i))
			}
			if stories.Less(j, i) {
				t.Error("Story #" + strconv.Itoa(i) + " should not be below Story #" + strconv.Itoa(j))
			}
		}
	}
	stories.Swap(0, 1)
	if stories[0] != story1 || stories[1] != story0 {
		t.Error("Story #0 should have been swapped with Story #1")
	}
	if stories[2] != story2 {
		t.Error("Story #2 should be equal to itself")
	}
}
