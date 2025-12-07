package cli

import "testing"

func TestParseEnvMergesAndIgnoresInvalid(t *testing.T) {
	base := map[string]string{"EXISTING": "1"}
	out := parseEnv([]string{"FOO=BAR", "BAD", "EMPTY="}, base)
	if out["FOO"] != "BAR" {
		t.Fatalf("expected FOO=BAR, got %v", out["FOO"])
	}
	if out["EXISTING"] != "1" {
		t.Fatalf("existing key lost")
	}
	if _, ok := out["BAD"]; ok {
		t.Fatalf("unexpected BAD entry")
	}
	if out["EMPTY"] != "" {
		t.Fatalf("expected EMPTY to be set to empty string, got %q", out["EMPTY"])
	}
}

func TestApplyManifestArgsSetsFields(t *testing.T) {
	opts := manifestOpts{}
	if err := applyManifestArgs([]string{"version", "0.0.1", "run", "python app.py", "details", "hi", "env", "FOO=BAR", "name", "other.json"}, &opts); err != nil {
		t.Fatalf("apply args: %v", err)
	}
	if opts.version != "0.0.1" || opts.run != "python app.py" || opts.details != "hi" || opts.name != "other.json" {
		t.Fatalf("unexpected opts %+v", opts)
	}
	found := false
	for _, e := range opts.env {
		if e == "FOO=BAR" {
			found = true
		}
	}
	if !found {
		t.Fatalf("env not set: %v", opts.env)
	}
}
