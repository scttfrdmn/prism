package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Tests for #730: a direct AMI mapping is checked against AWS. The placeholder validator accepted
// any ID and reported it as x86_64, so a bad mapping only failed at RunInstances and never fell
// back to script provisioning.

func mappedTemplate(strategy templates.AMIStrategy) *templates.Template {
	return &templates.Template{
		Name: "mapped",
		AMIConfig: templates.AMIConfig{
			Strategy:    strategy,
			AMIMappings: map[string]string{"us-west-2": "ami-0123456789abcdef0"},
		},
	}
}

func describeImagesReturning(images []ec2types.Image, err error) *MockEC2Client {
	return &MockEC2Client{DescribeImagesFunc: func(_ context.Context, in *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
		if err != nil {
			return nil, err
		}
		return &ec2.DescribeImagesOutput{Images: images}, nil
	}}
}

func TestResolveAMI_DirectMappingUsesAWSDetails(t *testing.T) {
	t.Setenv("PRISM_TEST_MODE", "")
	r := NewUniversalAMIResolver(describeImagesReturning([]ec2types.Image{{
		ImageId:      aws.String("ami-0123456789abcdef0"),
		Name:         aws.String("prism-python-ml-arm64"),
		Architecture: ec2types.ArchitectureValuesArm64,
		State:        ec2types.ImageStateAvailable,
		OwnerId:      aws.String("123456789012"),
		CreationDate: aws.String("2026-09-01T12:00:00.000Z"),
		Tags:         []ec2types.Tag{{Key: aws.String("prism:template"), Value: aws.String("python-ml")}},
	}}, nil))

	res, err := r.ResolveAMI(context.Background(), mappedTemplate(templates.AMIStrategyPreferred), "us-west-2")
	if err != nil {
		t.Fatalf("ResolveAMI: %v", err)
	}
	if res.ResolutionMethod != types.ResolutionDirectMapping || res.AMI == nil {
		t.Fatalf("method = %s, ami = %v; want a direct mapping", res.ResolutionMethod, res.AMI)
	}
	if res.AMI.Architecture != "arm64" || res.AMI.Name != "prism-python-ml-arm64" || res.AMI.Owner != "123456789012" {
		t.Errorf("AMI details not from AWS: %+v", res.AMI)
	}
	if res.AMI.CreationDate.IsZero() || res.AMI.Tags["prism:template"] != "python-ml" || res.AMI.Region != "us-west-2" {
		t.Errorf("creation date / tags / region missing: %+v", res.AMI)
	}
}

func TestResolveAMI_UnusableMappingFallsBackToScript(t *testing.T) {
	t.Setenv("PRISM_TEST_MODE", "")
	cases := map[string]*MockEC2Client{
		"not found (empty result)": describeImagesReturning(nil, nil),
		"not found (API error)":    describeImagesReturning(nil, errors.New("InvalidAMIID.NotFound")),
		"not available yet": describeImagesReturning([]ec2types.Image{{
			ImageId: aws.String("ami-0123456789abcdef0"), State: ec2types.ImageStatePending,
		}}, nil),
	}
	for name, client := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewUniversalAMIResolver(client)
			res, err := r.ResolveAMI(context.Background(), mappedTemplate(templates.AMIStrategyPreferred), "us-west-2")
			if err != nil {
				t.Fatalf("ResolveAMI: %v", err)
			}
			if res.ResolutionMethod != types.ResolutionFallbackScript || res.AMI != nil {
				t.Fatalf("method = %s, ami = %v; want script fallback (the mapping is unusable)", res.ResolutionMethod, res.AMI)
			}
			if res.Warning == "" {
				t.Error("fallback must tell the user why (Transparent Fallbacks)")
			}
		})
	}
}

func TestResolveAMI_RequiredStrategyRejectsMissingAMI(t *testing.T) {
	t.Setenv("PRISM_TEST_MODE", "")
	r := NewUniversalAMIResolver(describeImagesReturning(nil, nil))
	if _, err := r.ResolveAMI(context.Background(), mappedTemplate(templates.AMIStrategyRequired), "us-west-2"); err == nil {
		t.Fatal("ami_required with a missing AMI must fail resolution, not launch a bad ID")
	}
}

func TestValidateAMIExists_TestModeMakesNoAWSCall(t *testing.T) {
	t.Setenv("PRISM_TEST_MODE", "true")
	called := false
	r := NewUniversalAMIResolver(&MockEC2Client{DescribeImagesFunc: func(context.Context, *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
		called = true
		return &ec2.DescribeImagesOutput{}, nil
	}})
	if _, err := r.validateAMIExists(context.Background(), "ami-0123456789abcdef0"); err != nil {
		t.Fatalf("test mode: %v", err)
	}
	if called {
		t.Fatal("PRISM_TEST_MODE must return synthetic data without calling AWS")
	}
}

// A template whose AMI config no tier can satisfy (here: mapped only for a region with no fallback
// path back) must fall back to script provisioning. Cross-region search used to re-enter itself
// through the cyclic default region mapping and overflow the stack -- a fatal daemon crash.
func TestResolveAMI_NoTierMatchesDoesNotRecurse(t *testing.T) {
	t.Setenv("PRISM_TEST_MODE", "")
	tmpl := &templates.Template{Name: "elsewhere", AMIConfig: templates.AMIConfig{
		Strategy:    templates.AMIStrategyPreferred,
		AMIMappings: map[string]string{"ap-south-2": "ami-0123456789abcdef0"},
	}}
	r := NewUniversalAMIResolver(describeImagesReturning(nil, nil))

	res, err := r.ResolveAMI(context.Background(), tmpl, "us-west-2")
	if err != nil {
		t.Fatalf("ResolveAMI: %v", err)
	}
	if res.ResolutionMethod != types.ResolutionFallbackScript {
		t.Fatalf("method = %s, want script fallback", res.ResolutionMethod)
	}
}
