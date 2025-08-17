package daemon

import (
	"context"
)

// Key type for context values
type contextKey int

const (
	awsProfileKey contextKey = iota
	awsRegionKey
	apiVersionKey
)

// setAWSProfile adds the AWS profile to the context
func setAWSProfile(ctx context.Context, profile string) context.Context {
	return context.WithValue(ctx, awsProfileKey, profile)
}

// getAWSProfile gets the AWS profile from the context, or empty string if not set
func getAWSProfile(ctx context.Context) string {
	if profile, ok := ctx.Value(awsProfileKey).(string); ok {
		return profile
	}
	return ""
}

// setAWSRegion adds the AWS region to the context
func setAWSRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, awsRegionKey, region)
}

// getAWSRegion gets the AWS region from the context, or empty string if not set
func getAWSRegion(ctx context.Context) string {
	if region, ok := ctx.Value(awsRegionKey).(string); ok {
		return region
	}
	return ""
}

// setAPIVersion adds the API version to the context
func setAPIVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, apiVersionKey, version)
}
