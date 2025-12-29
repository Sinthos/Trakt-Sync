package trakt

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/zerolog/log"
)

const listItemsPageLimit = 100

// GetList retrieves a specific list
func (c *Client) GetList(username, listSlug string) (*List, error) {
	var list List
	user := url.PathEscape(username)
	slug := url.PathEscape(listSlug)
	path := fmt.Sprintf("/users/%s/lists/%s", user, slug)
	resp, err := c.doRequest("GET", path, nil, &list)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get list: %w", err)
	}
	return &list, nil
}

// GetListItems retrieves all items in a list
func (c *Client) GetListItems(username, listSlug string) ([]ListItem, error) {
	user := url.PathEscape(username)
	slug := url.PathEscape(listSlug)

	var allItems []ListItem
	page := 1

	for {
		var items []ListItem
		path := fmt.Sprintf("/users/%s/lists/%s/items?page=%d&limit=%d", user, slug, page, listItemsPageLimit)
		resp, err := c.doRequest("GET", path, nil, &items)
		if err != nil {
			return nil, fmt.Errorf("failed to get list items: %w", err)
		}

		allItems = append(allItems, items...)

		pageCount := parsePaginationPageCount(resp.Header)
		if pageCount == 0 || page >= pageCount {
			break
		}

		page++
	}

	return allItems, nil
}

// CreateList creates a new list
func (c *Client) CreateList(username string, req CreateListRequest) (*List, error) {
	var list List
	user := url.PathEscape(username)
	path := fmt.Sprintf("/users/%s/lists", user)
	_, err := c.doRequest("POST", path, req, &list)
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}
	log.Info().Str("list", req.Name).Msg("Created new list")
	return &list, nil
}

// AddItemsToList adds items to a list
func (c *Client) AddItemsToList(username, listSlug string, req AddToListRequest) error {
	user := url.PathEscape(username)
	slug := url.PathEscape(listSlug)
	path := fmt.Sprintf("/users/%s/lists/%s/items", user, slug)
	_, err := c.doRequest("POST", path, req, nil)
	if err != nil {
		return fmt.Errorf("failed to add items to list: %w", err)
	}
	return nil
}

// RemoveItemsFromList removes items from a list
func (c *Client) RemoveItemsFromList(username, listSlug string, req RemoveFromListRequest) error {
	user := url.PathEscape(username)
	slug := url.PathEscape(listSlug)
	path := fmt.Sprintf("/users/%s/lists/%s/items/remove", user, slug)
	_, err := c.doRequest("POST", path, req, nil)
	if err != nil {
		return fmt.Errorf("failed to remove items from list: %w", err)
	}
	return nil
}

// EnsureListExists checks if a list exists and creates it if it doesn't
func (c *Client) EnsureListExists(username, listSlug, listName, description, privacy string) error {
	list, err := c.GetList(username, listSlug)
	if err != nil {
		return err
	}

	if list == nil {
		if privacy == "" {
			privacy = "private"
		}
		_, err := c.CreateList(username, CreateListRequest{
			Name:           listName,
			Description:    description,
			Privacy:        privacy,
			DisplayNumbers: true,
			AllowComments:  false,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func parsePaginationPageCount(headers http.Header) int {
	value := headers.Get("X-Pagination-Page-Count")
	if value == "" {
		return 0
	}

	count, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return count
}
