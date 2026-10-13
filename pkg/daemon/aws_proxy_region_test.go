package daemon

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The /aws-proxy/ handler interpolates `region` into the reverse-proxy target host
// (https://<region>.console.aws.amazon.com). An unvalidated value such as "evil.com/#" pointed the
// proxy -- and any federation token -- at an arbitrary host (SSRF).
func TestAWSRegionPattern(t *testing.T) {
	for _, ok := range []string{"us-west-2", "us-east-1", "eu-central-1", "ap-southeast-2", "us-gov-west-1", "cn-north-1"} {
		assert.True(t, awsRegionPattern.MatchString(ok), ok)
	}
	for _, bad := range []string{"evil.com/#", "evil.com", "us-west-2.evil.com", "us-west-2/", "us-west-2#", "US-WEST-2", "us-west", "a-b-1@evil.com", "", "us-west-2\n"} {
		assert.False(t, awsRegionPattern.MatchString(bad), "%q must be rejected", bad)
	}
}

func TestAWSServiceProxy_RejectsHostInjectionInRegion(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	for _, region := range []string{"evil.com/#", "attacker.example", "us-west-2.evil.com"} {
		req := httptest.NewRequest(http.MethodGet, "/aws-proxy/s3/?region="+url.QueryEscape(region)+"&token=secret", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "region %q: body %s", region, w.Body.String())
		assert.Contains(t, w.Body.String(), "Invalid AWS region")
	}
}
