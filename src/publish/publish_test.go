package publish

import "testing"

func TestParseVersionLinesReadsGitOutput(t *testing.T) {
	out := `fe2944ace1692e3b62bad591a3f90dec0c174b32	refs/tags/v1.0.0
fe2944ace1692e3b62bad591a3f90dec0c174b32	refs/tags/v1.0.1
ignored refs/tags/releae
`
	versions := parseVersionLines(out)
	latest := latestVersion(versions)
	if latest.String() != "v1.0.1" {
		t.Fatalf("expected v1.0.1, got %s", latest.String())
	}
}

func TestBumpVersion(t *testing.T) {
	current, ok := parseVersion("v1.0.1")
	if !ok {
		t.Fatal("failed to parse version")
	}
	cases := map[string]string{
		"patch": "v1.0.2",
		"minor": "v1.1.0",
		"major": "v2.0.0",
	}
	for bump, want := range cases {
		if got := bumpVersion(current, bump).String(); got != want {
			t.Fatalf("bump %s: expected %s, got %s", bump, want, got)
		}
	}
}

func TestParseCommandPublishModes(t *testing.T) {
	cmd, err := parseCommand([]string{"minor", "--dry-run", "--allow-dirty"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.action != "publish" || cmd.bump != "minor" || !cmd.dryRun || !cmd.allowDirty {
		t.Fatalf("unexpected command: %+v", cmd)
	}

	cmd, err = parseCommand([]string{"--tag", "v1.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.action != "publish" || cmd.tag != "v1.0.1" {
		t.Fatalf("unexpected tag command: %+v", cmd)
	}
}
