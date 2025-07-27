package api

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
)

// User management API client methods

// ListUsers lists users with optional filtering
func (c *Client) ListUsers(filter *usermgmt.UserFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error) {
	var result usermgmt.PaginatedUsers
	
	// Build query parameters
	query := url.Values{}
	
	if filter != nil {
		if filter.Username != "" {
			query.Set("username", filter.Username)
		}
		
		if filter.Email != "" {
			query.Set("email", filter.Email)
		}
		
		if filter.Role != "" {
			query.Set("role", string(filter.Role))
		}
		
		if filter.Group != "" {
			query.Set("group", filter.Group)
		}
		
		if filter.Provider != "" {
			query.Set("provider", string(filter.Provider))
		}
		
		if filter.EnabledOnly {
			query.Set("enabled_only", "true")
		}
		
		if filter.DisabledOnly {
			query.Set("disabled_only", "true")
		}
	}
	
	if pagination != nil {
		if pagination.Page > 0 {
			query.Set("page", strconv.Itoa(pagination.Page))
		}
		
		if pagination.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(pagination.PageSize))
		}
		
		if pagination.SortBy != "" {
			query.Set("sort_by", pagination.SortBy)
		}
		
		if pagination.SortOrder != "" {
			query.Set("sort_order", pagination.SortOrder)
		}
	}
	
	path := "/api/v1/users"
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	
	err := c.get(path, &result)
	return &result, err
}

// GetUser gets a user by ID
func (c *Client) GetUser(id string) (*usermgmt.User, error) {
	var result usermgmt.User
	err := c.get(fmt.Sprintf("/api/v1/users/%s", id), &result)
	return &result, err
}

// GetUserByUsername gets a user by username
func (c *Client) GetUserByUsername(username string) (*usermgmt.User, error) {
	// Use filter to get user by username
	filter := &usermgmt.UserFilter{
		Username: username,
	}
	
	users, err := c.ListUsers(filter, nil)
	if err != nil {
		return nil, err
	}
	
	if len(users.Users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	
	return users.Users[0], nil
}

// CreateUser creates a new user
func (c *Client) CreateUser(user *usermgmt.User) (*usermgmt.User, error) {
	var result usermgmt.User
	err := c.post("/api/v1/users", user, &result)
	return &result, err
}

// UpdateUser updates an existing user
func (c *Client) UpdateUser(user *usermgmt.User) (*usermgmt.User, error) {
	var result usermgmt.User
	err := c.put(fmt.Sprintf("/api/v1/users/%s", user.ID), user, &result)
	return &result, err
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/users/%s", id))
}

// EnableUser enables a user
func (c *Client) EnableUser(id string) error {
	return c.post(fmt.Sprintf("/api/v1/users/%s/enable", id), nil, nil)
}

// DisableUser disables a user
func (c *Client) DisableUser(id string) error {
	return c.post(fmt.Sprintf("/api/v1/users/%s/disable", id), nil, nil)
}

// GetUserGroups gets the groups a user belongs to
func (c *Client) GetUserGroups(id string) ([]*usermgmt.Group, error) {
	var result []*usermgmt.Group
	err := c.get(fmt.Sprintf("/api/v1/users/%s/groups", id), &result)
	return result, err
}

// UpdateUserGroups updates the groups a user belongs to
func (c *Client) UpdateUserGroups(id string, groupNames []string) error {
	return c.put(fmt.Sprintf("/api/v1/users/%s/groups", id), groupNames, nil)
}

// ListGroups lists groups with optional filtering
func (c *Client) ListGroups(filter *usermgmt.GroupFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedGroups, error) {
	var result usermgmt.PaginatedGroups
	
	// Build query parameters
	query := url.Values{}
	
	if filter != nil {
		if filter.Name != "" {
			query.Set("name", filter.Name)
		}
		
		if filter.Provider != "" {
			query.Set("provider", string(filter.Provider))
		}
	}
	
	if pagination != nil {
		if pagination.Page > 0 {
			query.Set("page", strconv.Itoa(pagination.Page))
		}
		
		if pagination.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(pagination.PageSize))
		}
		
		if pagination.SortBy != "" {
			query.Set("sort_by", pagination.SortBy)
		}
		
		if pagination.SortOrder != "" {
			query.Set("sort_order", pagination.SortOrder)
		}
	}
	
	path := "/api/v1/groups"
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	
	err := c.get(path, &result)
	return &result, err
}

// GetGroup gets a group by ID
func (c *Client) GetGroup(id string) (*usermgmt.Group, error) {
	var result usermgmt.Group
	err := c.get(fmt.Sprintf("/api/v1/groups/%s", id), &result)
	return &result, err
}

// GetGroupByName gets a group by name
func (c *Client) GetGroupByName(name string) (*usermgmt.Group, error) {
	// Use filter to get group by name
	filter := &usermgmt.GroupFilter{
		Name: name,
	}
	
	groups, err := c.ListGroups(filter, nil)
	if err != nil {
		return nil, err
	}
	
	if len(groups.Groups) == 0 {
		return nil, fmt.Errorf("group not found")
	}
	
	return groups.Groups[0], nil
}

// CreateGroup creates a new group
func (c *Client) CreateGroup(group *usermgmt.Group) (*usermgmt.Group, error) {
	var result usermgmt.Group
	err := c.post("/api/v1/groups", group, &result)
	return &result, err
}

// UpdateGroup updates an existing group
func (c *Client) UpdateGroup(group *usermgmt.Group) (*usermgmt.Group, error) {
	var result usermgmt.Group
	err := c.put(fmt.Sprintf("/api/v1/groups/%s", group.ID), group, &result)
	return &result, err
}

// DeleteGroup deletes a group
func (c *Client) DeleteGroup(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/groups/%s", id))
}

// GetGroupUsers gets the users in a group
func (c *Client) GetGroupUsers(id string, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error) {
	var result usermgmt.PaginatedUsers
	
	// Build query parameters
	query := url.Values{}
	
	if pagination != nil {
		if pagination.Page > 0 {
			query.Set("page", strconv.Itoa(pagination.Page))
		}
		
		if pagination.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(pagination.PageSize))
		}
	}
	
	path := fmt.Sprintf("/api/v1/groups/%s/users", id)
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	
	err := c.get(path, &result)
	return &result, err
}

// UpdateGroupUsers updates the users in a group
func (c *Client) UpdateGroupUsers(id string, userIDs []string) error {
	return c.put(fmt.Sprintf("/api/v1/groups/%s/users", id), userIDs, nil)
}

// Authenticate authenticates a user with the given credentials
func (c *Client) Authenticate(username, password string) (*AuthenticationResult, error) {
	req := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}
	
	var result AuthenticationResult
	err := c.post("/api/v1/authenticate", req, &result)
	return &result, err
}

// AuthenticationResult represents the result of an authentication request
type AuthenticationResult struct {
	Token     string         `json:"token"`
	User      *usermgmt.User `json:"user"`
	ExpiresAt string         `json:"expires_at"`
}