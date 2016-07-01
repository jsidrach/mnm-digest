// Tests for the mnmdigest package
package mnmdigest

import (
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
	// TODO
}

// Test the stories implementation of sort.Interface
func TestStoriesSortInterface(t *testing.T) {
	// TODO
}
