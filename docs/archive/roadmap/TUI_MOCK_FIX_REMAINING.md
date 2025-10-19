# Remaining TUI Mock Client Fixes

**Status**: 3 of 4 mock clients fixed, 1 remaining

## Completed ✅

1. **instances_test.go** - mockAPIClient ✅
2. **dashboard_test.go** - mockAPIClientDashboard ✅
3. **instance_action_test.go** - instanceActionMockClient ✅

## Remaining ⚠️

### 1. profiles_test.go - mockAPIClientProfiles
**Error**: Missing `ApplyRightsizingRecommendation` and other methods

**Fix Required**: Add the following methods after the last existing method in mockAPIClientProfiles:

```go
func (m *mockAPIClientProfiles) ListProjects(ctx context.Context, filter *api.ProjectFilter) (*api.ListProjectsResponse, error) {
	return &api.ListProjectsResponse{}, nil
}

func (m *mockAPIClientProfiles) GetPolicyStatus(ctx context.Context) (*api.PolicyStatusResponse, error) {
	return &api.PolicyStatusResponse{}, nil
}

func (m *mockAPIClientProfiles) ListPolicySets(ctx context.Context) (*api.ListPolicySetsResponse, error) {
	return &api.ListPolicySetsResponse{}, nil
}

func (m *mockAPIClientProfiles) AssignPolicySet(ctx context.Context, policySetID string) error {
	return nil
}

func (m *mockAPIClientProfiles) SetPolicyEnforcement(ctx context.Context, enabled bool) error {
	return nil
}

func (m *mockAPIClientProfiles) CheckTemplateAccess(ctx context.Context, templateName string) (*api.TemplateAccessResponse, error) {
	return &api.TemplateAccessResponse{}, nil
}

func (m *mockAPIClientProfiles) ListMarketplaceTemplates(ctx context.Context, filter *api.MarketplaceFilter) (*api.ListMarketplaceTemplatesResponse, error) {
	return &api.ListMarketplaceTemplatesResponse{}, nil
}

func (m *mockAPIClientProfiles) ListMarketplaceCategories(ctx context.Context) (*api.ListCategoriesResponse, error) {
	return &api.ListCategoriesResponse{}, nil
}

func (m *mockAPIClientProfiles) ListMarketplaceRegistries(ctx context.Context) (*api.ListRegistriesResponse, error) {
	return &api.ListRegistriesResponse{}, nil
}

func (m *mockAPIClientProfiles) InstallMarketplaceTemplate(ctx context.Context, templateName string) error {
	return nil
}

func (m *mockAPIClientProfiles) ListAMIs(ctx context.Context) (*api.ListAMIsResponse, error) {
	return &api.ListAMIsResponse{}, nil
}

func (m *mockAPIClientProfiles) ListAMIBuilds(ctx context.Context) (*api.ListAMIBuildsResponse, error) {
	return &api.ListAMIBuildsResponse{}, nil
}

func (m *mockAPIClientProfiles) ListAMIRegions(ctx context.Context) (*api.ListAMIRegionsResponse, error) {
	return &api.ListAMIRegionsResponse{}, nil
}

func (m *mockAPIClientProfiles) DeleteAMI(ctx context.Context, amiID string) error {
	return nil
}

func (m *mockAPIClientProfiles) GetRightsizingRecommendations(ctx context.Context) (*api.GetRightsizingRecommendationsResponse, error) {
	return &api.GetRightsizingRecommendationsResponse{}, nil
}

func (m *mockAPIClientProfiles) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
	return nil
}

func (m *mockAPIClientProfiles) GetLogs(ctx context.Context, instanceName, logType string) (*api.LogsResponse, error) {
	return &api.LogsResponse{}, nil
}
```

### Other Files to Check

Also check these files for additional mock clients (may or may not need updates):
- `templates_test.go`
- `storage_test.go`
- `settings_test.go`
- `users_test.go`
- `repositories_test.go`

## Pattern for Adding Methods

1. Find the last method in the mock client type
2. Add all missing methods before the next test function or helper
3. Each method should:
   - Match the apiClient interface signature
   - Return empty response structs
   - Log calls if the mock tracks them
   - Return nil for errors unless shouldError is set

## Verification

After adding methods, compile with:
```bash
go test ./internal/tui/models/... 2>&1 | grep "does not implement"
```

If no output, all mocks are complete!

## Root Cause

The TUI uses a custom `apiClient` interface defined in `/internal/tui/models/common.go` that differs from the main `CloudWorkstationAPI` interface. As new features were added (Rightsizing, Policy, Marketplace, AMI, Logs), the interface grew but the test mocks weren't updated.

## Prevention

Consider adding a test that verifies all mock clients implement the full apiClient interface. Example:

```go
func TestMockClientsImplementInterface(t *testing.T) {
	var _ apiClient = (*mockAPIClient)(nil)
	var _ apiClient = (*mockAPIClientDashboard)(nil)
	var _ apiClient = (*instanceActionMockClient)(nil)
	var _ apiClient = (*mockAPIClientProfiles)(nil)
	// Add more as needed
}
```

This will cause compilation errors if mocks are incomplete.
