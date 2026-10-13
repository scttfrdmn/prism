package policy

import (
	"strings"
	"testing"
)

// Evaluators return (matches, reason): matches=true means the policy's restriction applies, so a
// deny-effect policy denies the request. Conditions arrive as a generic map; before, a decode
// failure was ignored and the policy became an empty one that never matched -- a malformed deny
// policy silently allowed everything. A policy whose conditions can't be decoded must now match.
func TestPolicyEvaluators_MalformedConditionsMatch(t *testing.T) {
	m := NewManager()
	req := &PolicyRequest{UserID: "u1", Action: "launch_template", Resource: "python-ml"}
	cases := map[string]struct {
		eval func(*Policy, *PolicyRequest) (bool, string)
		cond map[string]interface{}
	}{
		"template access": {m.evaluateTemplateAccessPolicy, map[string]interface{}{"denied_templates": "python-ml"}},
		"resource limits": {m.evaluateResourceLimitsPolicy, map[string]interface{}{"max_instances": "lots"}},
		"research user":   {m.evaluateResearchUserPolicy, map[string]interface{}{"max_users": "many"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			matches, reason := tc.eval(&Policy{ID: "p1", Conditions: tc.cond}, req)
			if !matches {
				t.Fatalf("malformed conditions did not match (reason %q): a deny policy would not apply", reason)
			}
			if !strings.Contains(reason, "Invalid policy conditions") {
				t.Errorf("reason = %q, want it to name the invalid conditions", reason)
			}
		})
	}
}

func evaluateWith(t *testing.T, conditions map[string]interface{}, resource string) *PolicyResponse {
	t.Helper()
	m := NewManager()
	set := &PolicySet{ID: "set1", Enabled: true, Policies: []*Policy{{
		ID: "deny-templates", Type: PolicyTypeTemplateAccess, Effect: PolicyEffectDeny,
		Enabled: true, Conditions: conditions,
	}}}
	if err := m.AddPolicySet(set); err != nil {
		t.Fatal(err)
	}
	if err := m.AssignPolicySet("u1", "set1"); err != nil {
		t.Fatal(err)
	}
	return m.EvaluatePolicy(&PolicyRequest{UserID: "u1", Action: "launch_template", Resource: resource})
}

// End to end: a malformed deny policy denies, instead of being skipped.
func TestEvaluatePolicy_MalformedDenyPolicyDenies(t *testing.T) {
	resp := evaluateWith(t, map[string]interface{}{"denied_templates": "python-ml"}, "python-ml")
	if resp.Allowed {
		t.Fatalf("malformed deny policy allowed the request: %+v", resp)
	}
	if !strings.Contains(resp.Reason, "Invalid policy conditions") {
		t.Errorf("reason = %q", resp.Reason)
	}
}

// Control: well-formed conditions still allow and deny as written.
func TestEvaluatePolicy_ValidDenyPolicy(t *testing.T) {
	deny := map[string]interface{}{"denied_templates": []interface{}{"python-ml"}}
	if resp := evaluateWith(t, deny, "python-ml"); resp.Allowed {
		t.Fatalf("denied template allowed: %+v", resp)
	}
	if resp := evaluateWith(t, deny, "r-research"); !resp.Allowed {
		t.Fatalf("template not on the deny list was denied: %+v", resp)
	}
}
