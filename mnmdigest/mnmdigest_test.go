// Tests for the mnmdigest package
package mnmdigest

import (
	"testing"
)

// Should initialize global variables and handlers
func TestInit(t *testing.T) {
}

// Should handle all requests
func TestHandleRequest(t *testing.T) {
}

// Should determine if the digest needs to be refreshed or not
func TestDigestNeedsRefresh(t *testing.T) {
}

// Should refresh the digest, storing the pages into the datastore
func TestRefreshDigest(t *testing.T) {
}

// Should fetch the new stories from menéame
func TestGetNewStories(t *testing.T) {
}

// Should filter the new stories, keeping only the unique ones, and returning a maximum of MaxStories
func TestFilterNewStories(t *testing.T) {
}

// Should update/delete the past stories
func TestUpdatePastStories(t *testing.T) {
}

// Should store the stories into the datastore
func TestStoreStories(t *testing.T) {
}

// Should generate the pages
func TestGeneratePages(t *testing.T) {
}

// Should get the digest
func TestGetDigest(t *testing.T) {
}

// Should store the digest
func TestStoreDigest(t *testing.T) {
}

// Should get the content of a tag
func TestGetTagContent(t *testing.T) {
}

// Test the stories implementation of sort.Interface
func TestStoriesSortInterface(t *testing.T) {
}
