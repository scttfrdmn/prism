package daemon

import (
	"context"
)

// Key type for context values
type contextKey int

const (
	awsProfileKey contextKey = iota
	awsRegionKey
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

// getAWSOptions creates AWS options from the context
func getAWSOptions(ctx context.Context) map[string]string {
	options := make(map[string]string)
	
	if profile := getAWSProfile(ctx); profile != "" {
		options["profile"] = profile
	}
	
	if region := getAWSRegion(ctx); region != "" {
		options["region"] = region
	}
	
	return options
}