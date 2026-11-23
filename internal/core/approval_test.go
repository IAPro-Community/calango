package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func mustDigest(t *testing.T, a Action) string {
	t.Helper()
	d, err := a.Digest()
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	return d
}

func TestDigestIgnoresKeyOrderAndWhitespace(t *testing.T) {
	a := Action{Kind: ActionSendEmail, Target: "ana@example.com", Args: json.RawMessage(`{"body":"x","subject":"Oi"}`)}
	b := Action{Kind: ActionSendEmail, Target: "ana@example.com", Args: json.RawMessage("{\n  \"subject\" : \"Oi\",\n  \"body\": \"x\"\n}")}
	if mustDigest(t, a) != mustDigest(t, b) {
		t.Fatal("equal actions with different key order or whitespace must share a digest")
	}
}

func TestDigestChangesWithTargetOrArgs(t *testing.T) {
	base := Action{Kind: ActionDelete, Target: "file.txt", Args: json.RawMessage(`{"force":false}`)}
	target := base
	target.Target = "other.txt"
	args := base
	args.Args = json.RawMessage(`{"force":true}`)
	if mustDigest(t, base) == mustDigest(t, target) {
		t.Fatal("changing the target must change the digest")
	}
	if mustDigest(t, base) == mustDigest(t, args) {
		t.Fatal("changing the arguments must change the digest")
	}
}

func TestNormalizeRejectsInvalidActions(t *testing.T) {
	cases := map[string]Action{
		"non-sensitive kind": {Kind: ActionKind("read_file"), Target: "a"},
		"empty target":       {Kind: ActionPay, Target: "   "},
		"args not object":    {Kind: ActionDeploy, Target: "prod", Args: json.RawMessage(`[1]`)},
		"trailing data":      {Kind: ActionMerge, Target: "main", Args: json.RawMessage(`{} {}`)},
	}
	for name, a := range cases {
		if _, err := a.Normalize(); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestNormalizeTreatsEmptyArgsAsEmptyObject(t *testing.T) {
	a := Action{Kind: ActionPublish, Target: "site"}
	n, err := a.Normalize()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if string(n.Args) != "{}" {
		t.Fatalf("args = %s, want {}", n.Args)
	}
	if n.Target != "site" {
		t.Fatalf("target = %q, want site", n.Target)
	}
}
