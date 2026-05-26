package publish

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorGreen  = "\033[38;5;76m"
	colorRed    = "\033[38;5;196m"
	colorYellow = "\033[38;5;214m"
	colorCyan   = "\033[38;5;44m"
	colorGray   = "\033[38;5;244m"
)

type target struct {
	goos   string
	goarch string
	ext    string
}

var releaseTargets = []target{
	{goos: "linux", goarch: "amd64"},
	{goos: "linux", goarch: "arm64"},
	{goos: "darwin", goarch: "amd64"},
	{goos: "darwin", goarch: "arm64"},
	{goos: "windows", goarch: "amd64", ext: ".exe"},
}

type config struct {
	root       string
	dist       string
	repo       string
	token      string
	apiBase    string
	dryRun     bool
	allowDirty bool
	out        io.Writer
	errOut     io.Writer
}

type command struct {
	action     string
	bump       string
	tag        string
	repo       string
	token      string
	dist       string
	dryRun     bool
	allowDirty bool
}

type version struct {
	major int
	minor int
	patch int
	raw   string
}

type release struct {
	ID        int64     `json:"id"`
	TagName   string    `json:"tag_name"`
	Name      string    `json:"name"`
	UploadURL string    `json:"upload_url"`
	CreatedAt time.Time `json:"created_at"`
	Assets    []asset   `json:"assets"`
}

type asset struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type gitRef struct {
	Ref    string `json:"ref"`
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

type githubError struct {
	status int
	body   string
}

func (e githubError) Error() string {
	body := strings.TrimSpace(e.body)
	if body == "" {
		return fmt.Sprintf("github api returned status %d", e.status)
	}
	return fmt.Sprintf("github api returned status %d: %s", e.status, body)
}

func Run(args []string) error {
	return RunWithWriters(args, os.Stdout, os.Stderr)
}

func RunWithWriters(args []string, out io.Writer, errOut io.Writer) error {
	cmd, err := parseCommand(args)
	if err != nil {
		return err
	}
	if cmd.action == "help" {
		PrintUsage(out)
		return nil
	}

	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	cfg, err := loadConfig(root, cmd, out, errOut)
	if err != nil {
		return err
	}

	switch cmd.action {
	case "list":
		return listVersions(context.Background(), cfg)
	case "status":
		return printStatus(context.Background(), cfg)
	case "build":
		return buildOnly(context.Background(), cfg)
	case "publish":
		return publishRelease(context.Background(), cfg, cmd)
	default:
		PrintUsage(out)
		return fmt.Errorf("unknown publish action %q", cmd.action)
	}
}

func PrintUsage(out io.Writer) {
	fmt.Fprintf(out, `
  %sZPM Publish%s

  %sUsage:%s
    zpm publish --list
    zpm publish status
    zpm publish build
    zpm publish patch|minor|major [--dry-run] [--allow-dirty]
    zpm publish --tag v1.0.1 [--dry-run] [--allow-dirty]

  %sEnvironment:%s
    GITHUB_TOKEN or GH_TOKEN        Token with contents:write for publishing
    GITHUB_REPOSITORY               Repository slug, for example shellhaki/zpm

  %s.env is loaded from the repository root when present.%s
`, colorBold, colorReset, colorYellow, colorReset, colorCyan, colorReset, colorDim, colorReset)
}

func parseCommand(args []string) (command, error) {
	cmd := command{action: "help"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			cmd.action = "help"
		case arg == "--list" || arg == "list" || arg == "versions":
			cmd.action = "list"
		case arg == "status":
			cmd.action = "status"
		case arg == "build":
			cmd.action = "build"
		case arg == "patch" || arg == "minor" || arg == "major":
			cmd.action = "publish"
			cmd.bump = arg
		case strings.HasPrefix(arg, "v") && looksLikeVersion(arg):
			cmd.action = "publish"
			cmd.tag = arg
		case arg == "--dry-run":
			cmd.dryRun = true
		case arg == "--allow-dirty":
			cmd.allowDirty = true
		case arg == "--repo":
			if i+1 >= len(args) {
				return cmd, errors.New("missing value for --repo")
			}
			cmd.repo = args[i+1]
			i++
		case strings.HasPrefix(arg, "--repo="):
			cmd.repo = strings.TrimPrefix(arg, "--repo=")
		case arg == "--token":
			if i+1 >= len(args) {
				return cmd, errors.New("missing value for --token")
			}
			cmd.token = args[i+1]
			i++
		case strings.HasPrefix(arg, "--token="):
			cmd.token = strings.TrimPrefix(arg, "--token=")
		case arg == "--tag":
			if i+1 >= len(args) {
				return cmd, errors.New("missing value for --tag")
			}
			cmd.action = "publish"
			cmd.tag = args[i+1]
			i++
		case strings.HasPrefix(arg, "--tag="):
			cmd.action = "publish"
			cmd.tag = strings.TrimPrefix(arg, "--tag=")
		case arg == "--dist":
			if i+1 >= len(args) {
				return cmd, errors.New("missing value for --dist")
			}
			cmd.dist = args[i+1]
			i++
		case strings.HasPrefix(arg, "--dist="):
			cmd.dist = strings.TrimPrefix(arg, "--dist=")
		default:
			return cmd, fmt.Errorf("unknown publish argument %q", arg)
		}
	}
	return cmd, nil
}

func loadConfig(root string, cmd command, out io.Writer, errOut io.Writer) (config, error) {
	env := loadDotEnv(filepath.Join(root, ".env"))
	value := func(keys ...string) string {
		for _, key := range keys {
			if v := strings.TrimSpace(os.Getenv(key)); v != "" {
				return v
			}
			if v := strings.TrimSpace(env[key]); v != "" {
				return v
			}
		}
		return ""
	}

	repo := firstNonEmpty(cmd.repo, value("GITHUB_REPOSITORY", "GITHUB_REPO", "ZPM_GITHUB_REPO", "ZPM_REPO"))
	if repo == "" {
		repo = inferRepoFromGit(root)
	}
	repo = normalizeRepo(repo)

	dist := cmd.dist
	if dist == "" {
		dist = filepath.Join(root, "dist")
	}
	if !filepath.IsAbs(dist) {
		dist = filepath.Join(root, dist)
	}

	cfg := config{
		root:       root,
		dist:       dist,
		repo:       repo,
		token:      firstNonEmpty(cmd.token, value("GITHUB_TOKEN", "GH_TOKEN")),
		apiBase:    firstNonEmpty(value("GITHUB_API_URL"), "https://api.github.com"),
		dryRun:     cmd.dryRun || truthy(value("ZPM_PUBLISH_DRY_RUN")),
		allowDirty: cmd.allowDirty || truthy(value("ZPM_PUBLISH_ALLOW_DIRTY")),
		out:        out,
		errOut:     errOut,
	}
	if cfg.repo == "" {
		return cfg, errors.New("github repo not found; set GITHUB_REPOSITORY=shellhaki/zpm in .env")
	}
	return cfg, nil
}

func listVersions(ctx context.Context, cfg config) error {
	fmt.Fprintf(cfg.out, "\n  %sZPM release versions%s %s%s%s\n\n", colorBold, colorReset, colorGray, cfg.repo, colorReset)

	versions, err := collectVersions(ctx, cfg)
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		fmt.Fprintf(cfg.out, "  %sNo versions found.%s\n", colorDim, colorReset)
		return nil
	}
	sortVersionsDesc(versions)
	for _, v := range versions {
		fmt.Fprintf(cfg.out, "  %s%s%s\n", colorGreen, v.raw, colorReset)
	}
	return nil
}

func printStatus(ctx context.Context, cfg config) error {
	versions, err := collectVersions(ctx, cfg)
	if err != nil {
		return err
	}
	latest := latestVersion(versions)
	fmt.Fprintf(cfg.out, "\n  %sZPM publish status%s\n\n", colorBold, colorReset)
	fmt.Fprintf(cfg.out, "  Repo:      %s\n", cfg.repo)
	fmt.Fprintf(cfg.out, "  Root:      %s\n", cfg.root)
	fmt.Fprintf(cfg.out, "  Dist:      %s\n", cfg.dist)
	if cfg.token == "" {
		fmt.Fprintf(cfg.out, "  Token:     %smissing%s\n", colorYellow, colorReset)
	} else {
		fmt.Fprintf(cfg.out, "  Token:     %sloaded%s\n", colorGreen, colorReset)
	}
	if latest.raw == "" {
		fmt.Fprintf(cfg.out, "  Latest:    none\n")
		fmt.Fprintf(cfg.out, "  Next:      patch v0.0.1, minor v0.1.0, major v1.0.0\n")
		return nil
	}
	fmt.Fprintf(cfg.out, "  Latest:    %s\n", latest.raw)
	fmt.Fprintf(cfg.out, "  Next:      patch %s, minor %s, major %s\n", bumpVersion(latest, "patch").String(), bumpVersion(latest, "minor").String(), bumpVersion(latest, "major").String())
	return nil
}

func buildOnly(ctx context.Context, cfg config) error {
	step := newStepper(cfg.out, 3)
	if err := step.Run("running release tests", func() error {
		return runTests(ctx, cfg)
	}); err != nil {
		return err
	}
	artifacts, err := buildArtifactsWithStep(ctx, cfg, step)
	if err != nil {
		return err
	}
	if err := step.Run("summarizing artifacts", func() error {
		for _, artifact := range artifacts {
			info, statErr := os.Stat(artifact)
			if statErr != nil {
				return statErr
			}
			fmt.Fprintf(cfg.out, "    %s %s\n", filepath.Base(artifact), byteSize(info.Size()))
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func publishRelease(ctx context.Context, cfg config, cmd command) error {
	if cfg.token == "" && !cfg.dryRun {
		return errors.New("GITHUB_TOKEN or GH_TOKEN is required for zpm publish")
	}

	step := newStepper(cfg.out, 8)
	var tag string
	var head string
	var artifacts []string
	var rel release

	if err := step.Run("checking git repository", func() error {
		var err error
		head, err = gitOutput(cfg.root, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if cfg.allowDirty {
			return nil
		}
		dirty, err := gitOutput(cfg.root, "status", "--porcelain")
		if err != nil {
			return err
		}
		if strings.TrimSpace(dirty) != "" {
			return errors.New("working tree has changes; commit them or pass --allow-dirty")
		}
		return nil
	}); err != nil {
		return err
	}

	if err := step.Run("resolving release version", func() error {
		if cmd.tag != "" {
			if !looksLikeVersion(cmd.tag) {
				return fmt.Errorf("invalid release tag %q", cmd.tag)
			}
			tag = ensureVPrefix(cmd.tag)
			return nil
		}
		versions, err := collectVersions(ctx, cfg)
		if err != nil {
			return err
		}
		next := bumpVersion(latestVersion(versions), cmd.bump)
		tag = next.String()
		return nil
	}); err != nil {
		return err
	}
	fmt.Fprintf(cfg.out, "    release tag: %s%s%s\n", colorCyan, tag, colorReset)

	if err := step.Run("running release tests", func() error {
		return runTests(ctx, cfg)
	}); err != nil {
		return err
	}

	artifacts, err := buildArtifactsWithStep(ctx, cfg, step)
	if err != nil {
		return err
	}

	if err := step.Run("creating local git tag", func() error {
		if cfg.dryRun {
			fmt.Fprintf(cfg.out, "    dry run: would create local tag %s\n", tag)
			return nil
		}
		return ensureLocalTag(cfg.root, tag, head)
	}); err != nil {
		return err
	}

	if err := step.Run("creating remote git tag", func() error {
		if cfg.dryRun {
			fmt.Fprintf(cfg.out, "    dry run: would create refs/tags/%s at %s\n", tag, shortSHA(head))
			return nil
		}
		client := githubClient{cfg: cfg, http: http.DefaultClient}
		return client.ensureRemoteTag(ctx, tag, head)
	}); err != nil {
		return err
	}

	if err := step.Run("creating github release", func() error {
		if cfg.dryRun {
			fmt.Fprintf(cfg.out, "    dry run: would create release %s\n", tag)
			rel = release{TagName: tag}
			return nil
		}
		client := githubClient{cfg: cfg, http: http.DefaultClient}
		var err error
		rel, err = client.ensureRelease(ctx, tag, head)
		return err
	}); err != nil {
		return err
	}

	if err := step.Run("uploading release assets", func() error {
		if cfg.dryRun {
			for _, artifact := range artifacts {
				fmt.Fprintf(cfg.out, "    dry run: would upload %s\n", filepath.Base(artifact))
			}
			return nil
		}
		client := githubClient{cfg: cfg, http: http.DefaultClient}
		for _, artifact := range artifacts {
			if err := client.uploadAsset(ctx, rel, artifact); err != nil {
				return err
			}
			fmt.Fprintf(cfg.out, "    uploaded %s\n", filepath.Base(artifact))
		}
		return nil
	}); err != nil {
		return err
	}

	if cfg.dryRun {
		fmt.Fprintf(cfg.out, "\n  %s[done]%s dry run completed for %s\n", colorGreen, colorReset, tag)
		fmt.Fprintf(cfg.out, "  No tag, release, or assets were created.\n\n")
		return nil
	}
	fmt.Fprintf(cfg.out, "\n  %s[done]%s release %s is ready\n", colorGreen, colorReset, tag)
	fmt.Fprintf(cfg.out, "  https://github.com/%s/releases/tag/%s\n\n", cfg.repo, tag)
	return nil
}

func buildArtifactsWithStep(ctx context.Context, cfg config, step *stepper) ([]string, error) {
	var artifacts []string
	err := step.Run("building cross-platform binaries", func() error {
		var err error
		artifacts, err = buildArtifacts(ctx, cfg)
		return err
	})
	return artifacts, err
}

func runTests(ctx context.Context, cfg config) error {
	if err := runCommand(ctx, cfg, filepath.Join(cfg.root, "daemon"), "go", "test", "./..."); err != nil {
		return err
	}
	return runCommand(ctx, cfg, filepath.Join(cfg.root, "src"), "go", "test", "./...")
}

func buildArtifacts(ctx context.Context, cfg config) ([]string, error) {
	if err := os.RemoveAll(cfg.dist); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.dist, 0755); err != nil {
		return nil, err
	}

	artifacts := make([]string, 0, len(releaseTargets))
	for _, target := range releaseTargets {
		name := fmt.Sprintf("zpm-%s-%s", target.goos, target.goarch)
		dir := filepath.Join(cfg.dist, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		fmt.Fprintf(cfg.out, "    building %s/%s\n", target.goos, target.goarch)
		env := []string{
			"GOOS=" + target.goos,
			"GOARCH=" + target.goarch,
			"CGO_ENABLED=0",
		}
		if err := runCommandWithEnv(ctx, cfg, filepath.Join(cfg.root, "src"), env, "go", "build", "-trimpath", "-o", filepath.Join(dir, "zpm"+target.ext), "."); err != nil {
			return nil, err
		}
		if err := runCommandWithEnv(ctx, cfg, filepath.Join(cfg.root, "daemon"), env, "go", "build", "-trimpath", "-o", filepath.Join(dir, "zpmd"+target.ext), "."); err != nil {
			return nil, err
		}
		if err := copyFile(filepath.Join(cfg.root, "README.md"), filepath.Join(dir, "README.md")); err != nil {
			return nil, err
		}

		artifact := filepath.Join(cfg.dist, name+".tar.gz")
		if err := tarGz(artifact, cfg.dist, name); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func ensureLocalTag(root string, tag string, head string) error {
	existing, err := gitOutput(root, "rev-parse", "--verify", "refs/tags/"+tag)
	if err == nil {
		if strings.TrimSpace(existing) != strings.TrimSpace(head) {
			return fmt.Errorf("local tag %s already points to %s, not %s", tag, shortSHA(existing), shortSHA(head))
		}
		return nil
	}
	return runSimple(root, "git", "tag", tag)
}

type githubClient struct {
	cfg  config
	http *http.Client
}

func (c githubClient) listReleases(ctx context.Context) ([]release, error) {
	var releases []release
	err := c.doJSON(ctx, http.MethodGet, "/releases?per_page=100", nil, &releases)
	return releases, err
}

func (c githubClient) getReleaseByTag(ctx context.Context, tag string) (release, error) {
	var rel release
	err := c.doJSON(ctx, http.MethodGet, "/releases/tags/"+url.PathEscape(tag), nil, &rel)
	return rel, err
}

func (c githubClient) ensureRemoteTag(ctx context.Context, tag string, head string) error {
	path := "/git/ref/tags/" + url.PathEscape(tag)
	var ref gitRef
	err := c.doJSON(ctx, http.MethodGet, path, nil, &ref)
	if err == nil {
		if ref.Object.SHA != "" && ref.Object.SHA != head {
			return fmt.Errorf("remote tag %s points to %s, not %s", tag, shortSHA(ref.Object.SHA), shortSHA(head))
		}
		return nil
	}
	var ghErr githubError
	if !errors.As(err, &ghErr) || ghErr.status != http.StatusNotFound {
		return err
	}

	body := map[string]string{
		"ref": "refs/tags/" + tag,
		"sha": head,
	}
	return c.doJSON(ctx, http.MethodPost, "/git/refs", body, &ref)
}

func (c githubClient) ensureRelease(ctx context.Context, tag string, head string) (release, error) {
	existing, err := c.getReleaseByTag(ctx, tag)
	if err == nil {
		return existing, nil
	}
	var ghErr githubError
	if !errors.As(err, &ghErr) || ghErr.status != http.StatusNotFound {
		return release{}, err
	}

	body := map[string]any{
		"tag_name":         tag,
		"target_commitish": head,
		"name":             tag,
		"body":             "ZPM release " + tag,
		"draft":            false,
		"prerelease":       false,
	}
	var rel release
	err = c.doJSON(ctx, http.MethodPost, "/releases", body, &rel)
	return rel, err
}

func (c githubClient) uploadAsset(ctx context.Context, rel release, path string) error {
	for _, existing := range rel.Assets {
		if existing.Name == filepath.Base(path) {
			if err := c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/releases/assets/%d", existing.ID), nil, nil); err != nil {
				return err
			}
			break
		}
	}

	uploadURL := rel.UploadURL
	if before, _, found := strings.Cut(uploadURL, "{"); found {
		uploadURL = before
	}
	if uploadURL == "" {
		uploadURL = fmt.Sprintf("https://uploads.github.com/repos/%s/releases/%d/assets", c.cfg.repo, rel.ID)
	}
	parsed, err := url.Parse(uploadURL)
	if err != nil {
		return err
	}
	query := parsed.Query()
	query.Set("name", filepath.Base(path))
	parsed.RawQuery = query.Encode()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), file)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.token)
	req.Header.Set("Content-Type", "application/gzip")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return githubError{status: res.StatusCode, body: string(body)}
	}
	return nil
}

func (c githubClient) doJSON(ctx context.Context, method string, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		pr, pw := io.Pipe()
		go func() {
			err := json.NewEncoder(pw).Encode(in)
			pw.CloseWithError(err)
		}()
		body = pr
	}

	u := c.cfg.apiBase + "/repos/" + c.cfg.repo + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return githubError{status: res.StatusCode, body: string(data)}
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func collectVersions(ctx context.Context, cfg config) ([]version, error) {
	var versions []version
	var firstErr error

	client := githubClient{cfg: cfg, http: http.DefaultClient}
	releases, err := client.listReleases(ctx)
	if err == nil {
		for _, rel := range releases {
			if v, ok := parseVersion(rel.TagName); ok {
				versions = append(versions, v)
			}
		}
	} else {
		firstErr = err
	}

	for _, source := range []func(string) ([]version, error){localGitTags, remoteGitTags} {
		tags, tagErr := source(cfg.root)
		if tagErr != nil {
			if firstErr == nil {
				firstErr = tagErr
			}
			continue
		}
		versions = append(versions, tags...)
	}

	versions = dedupeVersions(versions)
	if len(versions) > 0 {
		return versions, nil
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return versions, nil
}

func localGitTags(root string) ([]version, error) {
	out, err := gitOutput(root, "tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	return parseVersionLines(out), nil
}

func remoteGitTags(root string) ([]version, error) {
	out, err := gitOutput(root, "ls-remote", "--tags", "origin", "refs/tags/v*")
	if err != nil {
		return nil, err
	}
	return parseVersionLines(out), nil
}

func parseVersionLines(out string) []version {
	var versions []version
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) > 0 {
			line = fields[len(fields)-1]
		}
		line = strings.TrimSuffix(line, "^{}")
		if v, ok := parseVersion(line); ok {
			versions = append(versions, v)
		}
	}
	return versions
}

func parseVersion(raw string) (version, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "refs/tags/")
	raw = ensureVPrefix(raw)
	if !looksLikeVersion(raw) {
		return version{}, false
	}
	parts := strings.Split(strings.TrimPrefix(raw, "v"), ".")
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return version{major: major, minor: minor, patch: patch, raw: fmt.Sprintf("v%d.%d.%d", major, minor, patch)}, true
}

func looksLikeVersion(raw string) bool {
	raw = strings.TrimPrefix(strings.TrimSpace(raw), "v")
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func ensureVPrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "v") {
		return raw
	}
	return "v" + raw
}

func latestVersion(versions []version) version {
	if len(versions) == 0 {
		return version{}
	}
	sortVersionsDesc(versions)
	return versions[0]
}

func bumpVersion(v version, bump string) version {
	switch bump {
	case "major":
		return version{major: v.major + 1, raw: ""}
	case "minor":
		return version{major: v.major, minor: v.minor + 1, raw: ""}
	default:
		return version{major: v.major, minor: v.minor, patch: v.patch + 1, raw: ""}
	}
}

func (v version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
}

func sortVersionsDesc(versions []version) {
	sort.Slice(versions, func(i, j int) bool {
		a := versions[i]
		b := versions[j]
		if a.major != b.major {
			return a.major > b.major
		}
		if a.minor != b.minor {
			return a.minor > b.minor
		}
		return a.patch > b.patch
	})
}

func dedupeVersions(versions []version) []version {
	seen := map[string]bool{}
	out := []version{}
	for _, v := range versions {
		key := v.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		v.raw = key
		out = append(out, v)
	}
	return out
}

type stepper struct {
	out   io.Writer
	total int
	index int
}

func newStepper(out io.Writer, total int) *stepper {
	return &stepper{out: out, total: total}
}

func (s *stepper) Run(label string, fn func() error) error {
	s.index++
	fmt.Fprintf(s.out, "\n  %s[%d/%d]%s %s\n", colorCyan, s.index, s.total, colorReset, label)
	err := fn()
	if err != nil {
		fmt.Fprintf(s.out, "  %s[failed]%s %s\n", colorRed, colorReset, label)
		return err
	}
	fmt.Fprintf(s.out, "  %s[ok]%s %s\n", colorGreen, colorReset, label)
	return nil
}

func runCommand(ctx context.Context, cfg config, dir string, name string, args ...string) error {
	return runCommandWithEnv(ctx, cfg, dir, nil, name, args...)
}

func runCommandWithEnv(ctx context.Context, cfg config, dir string, extraEnv []string, name string, args ...string) error {
	fmt.Fprintf(cfg.out, "    > %s %s\n", name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = cfg.out
	cmd.Stderr = cfg.errOut
	return cmd.Run()
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	data, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exit.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func runSimple(root string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	if data, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), strings.TrimSpace(string(data)))
	}
	return nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if fileExists(filepath.Join(dir, "src", "go.mod")) && fileExists(filepath.Join(dir, "daemon", "go.mod")) && fileExists(filepath.Join(dir, "README.md")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = wd
	data, err := cmd.Output()
	if err != nil {
		return "", errors.New("could not find zpm repository root")
	}
	return strings.TrimSpace(string(data)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func inferRepoFromGit(root string) string {
	out, err := gitOutput(root, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return normalizeRepo(out)
}

func normalizeRepo(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ".git")
	value = strings.TrimPrefix(value, "git@github.com:")
	value = strings.TrimPrefix(value, "https://github.com/")
	value = strings.TrimPrefix(value, "http://github.com/")
	if strings.Count(value, "/") != 1 {
		return ""
	}
	return value
}

func loadDotEnv(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			} else if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) || (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
				value = strings.Trim(value, "'\"")
			}
		}
		if key != "" {
			env[key] = value
		}
	}
	return env
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func tarGz(dst string, baseDir string, dirName string) error {
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	root := filepath.Join(baseDir, dirName)
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tw, file)
		return err
	})
}

func byteSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
