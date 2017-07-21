package main

/** Create catalog entry from build options
Assumptions -  build number will be unique within a branch

Directoy layout.
base/                        # Go Templates (Create/Edit These)
	|_ catalogIcon.(png|svg)   # Copy for branch/catalogIcon.(png|svg)
	|_ config.tmpl             # Template for branch/config.yml
	|_ rancher-compose.tmpl    # Template for branch/0/rancher-compose.yml
	|_ docker-compose.tmpl     # Template for branch/0/docker-compose.yml
templates/
	|_ <branch>
		|_ catalogIcon.(png|svg)
		|_ config.yml
			|_ <build number>
				|_ rancher-compose.yml
				|_ docker-composer.yml

tags:
  regex
  first non-latest tag
  latest
branch and repo name:
  lowercase everything
  turn _\.\s into -
  if branch is PLUGIN_RELEASE_BRANCH use repo name
**/

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Sirupsen/logrus"
)

const (
	baseDir                    string = "rancher-catalog"
	repoDir                    string = "rancher-catalog/repo"
	templateDir                string = "base"
	dockerComposeTemplateFile  string = "docker-compose.tmpl"
	rancherComposeTemplateFile string = "rancher-compose.tmpl"
	configTemplateFile         string = "config.tmpl"
	iconFileBase               string = "catalogIcon"
)

type (
	// Catalog - Rancher catalog repository.
	Catalog struct {
		Context       string
		Repo          string
		ReleaseBranch string
	}
	// Docker - Docker Hub parameters.
	Docker struct {
		Repo string
	}
	// Github - Login parameters.
	Github struct {
		Username string
		Token    string
		Email    string
	}
	// Build - Parameters from Drone.
	Build struct {
		Context string   // Drone build context
		Tags    []string // Drone build tags
		Repo    string   // Drone build repo
		Branch  string   // Drone build repository
		Number  int64    // Drone build number
	}
	// Plugin defines the Docker plugin parameters.
	Plugin struct {
		Build    Build   // Drone Build details
		Catalog  Catalog // Rancher catalog
		Docker   Docker
		Dryrun   bool // Skip catalog push
		Debug    bool
		Github   Github // Github creds to get rancher catalog
		TagRegex string
	}
	// TemplateContext - the values that get passed into the template.
	TemplateContext struct {
		Tag        string
		Build      int64
		Project    string
		GithubRepo string
		DockerRepo string
		Branch     string
	}
)

// Exec executes the plugin step
func (p Plugin) Exec() error {
	if p.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	// Set HOME to catalog context.
	// So git can write its config while testing and not stomp on yours.
	os.Setenv("HOME", p.Catalog.Context)

	var (
		err             error
		templateContext TemplateContext
	)

	// Set up template parameters.
	templateContext.Build = p.Build.Number
	templateContext.GithubRepo = p.Build.Repo
	templateContext.DockerRepo = p.Docker.Repo
	templateContext.Project = fixName(p.Build.Repo)
	templateContext.Branch = fixName(p.Build.Branch)
	templateContext.Tag, err = pickTag(p.Build.Tags, p.TagRegex)
	if err != nil {
		return err
	}
	logrus.Infof("Using Tag: %s", templateContext.Tag)

	// If this is the release branch (master) then set the dir name to the project name
	if p.Build.Branch == p.Catalog.ReleaseBranch {
		logrus.Infof("Release Branch detected. Useing Project: '%s' as .Branch variable", templateContext.Project)
		templateContext.Branch = templateContext.Project
	}
	branchDir := fmt.Sprintf("./templates/%s", templateContext.Branch)
	buildDir := fmt.Sprintf("%s/%d", branchDir, p.Build.Number)

	// Clear out old repo
	os.Mkdir(p.Catalog.Context, 0755)
	os.RemoveAll(fmt.Sprintf("%s%s", p.Catalog.Context, repoDir))

	// Setup git
	err = execute(gitConfigureEmail(p), p)
	if err != nil {
		return err
	}
	err = execute(gitConfigureUser(p), p)
	if err != nil {
		return err
	}
	gitCredsPath := fmt.Sprintf("%s%s", p.Catalog.Context, "/.git-credentials")
	err = writeGitCredentials(p.Github, gitCredsPath)
	if err != nil {
		return err
	}
	err = execute(gitConfigureCredentials(), p)
	if err != nil {
		return err
	}

	// Clone repo
	err = execute(cloneCatalogRepo(p), p)
	if err != nil {
		return err
	}
	os.Chdir(fmt.Sprintf("%s%s", p.Catalog.Context, repoDir))

	// Parse templates
	dockerComposeTmpl, err := parseTemplateFile(fmt.Sprintf("./%s/%s", templateDir, dockerComposeTemplateFile))
	if err != nil {
		return err
	}
	rancherComposeTmpl, err := parseTemplateFile(fmt.Sprintf("./%s/%s", templateDir, rancherComposeTemplateFile))
	if err != nil {
		return err
	}
	configTmpl, err := parseTemplateFile(fmt.Sprintf("./%s/%s", templateDir, configTemplateFile))
	if err != nil {
		return err
	}

	// Create catalog dir
	logrus.Infof("Creating Catalog Entries for: %s/%d", p.Build.Branch, p.Build.Number)
	err = os.MkdirAll(buildDir, 0755)
	if err != nil {
		return err
	}

	// Copy Icon file to branchDir
	var cpIcon *exec.Cmd
	iconFilePath := fmt.Sprintf("./%s/%s", templateDir, iconFileBase)
	cpIcon, err = copyIcon(iconFilePath, branchDir)
	if err != nil {
		return err
	}
	err = execute(cpIcon, p)
	if err != nil {
		return err
	}

	// Create catalog entries
	configTarget := fmt.Sprintf("%s/config.yml", branchDir)
	dockerComposeTarget := fmt.Sprintf("%s/docker-compose.yml", buildDir)
	rancherComposeTarget := fmt.Sprintf("%s/rancher-compose.yml", buildDir)
	err = executeTemplate(configTarget, configTmpl, templateContext)
	if err != nil {
		return err
	}
	err = executeTemplate(dockerComposeTarget, dockerComposeTmpl, templateContext)
	if err != nil {
		return err
	}
	err = executeTemplate(rancherComposeTarget, rancherComposeTmpl, templateContext)
	if err != nil {
		return err
	}

	// Commit changes.
	if p.Dryrun == false {
		if gitChanged() {
			err = execute(addCatalogRepo(), p)
			if err != nil {
				return err
			}
			err = execute(commitCatalogRepo(p), p)
			if err != nil {
				return err
			}
			err = execute(pushCatalogRepo(), p)
			if err != nil {
				return err
			}
		}
	} else {
		logrus.Info("Dryrun true - Skipping commit and push")
	}
	return nil
}

func gitConfigureEmail(p Plugin) *exec.Cmd {
	return exec.Command("git", "config", "--global", "user.email", p.Github.Email)
}

func gitConfigureUser(p Plugin) *exec.Cmd {
	return exec.Command("git", "config", "--global", "user.name", p.Github.Username)
}

func gitConfigureCredentials() *exec.Cmd {
	return exec.Command("git", "config", "--global", "credential.helper", "store")
}

func writeGitCredentials(github Github, path string) error {
	logrus.Debug("Creating ", path)
	targetFile, err := os.Create(path)
	if err != nil {
		return err
	}
	tmpl, err := template.New("git-credentials").Parse("https://{{ .Token }}:@github.com\n")
	if err != nil {
		return err
	}
	err = tmpl.Execute(targetFile, github)
	targetFile.Close()
	return err
}

func cloneCatalogRepo(p Plugin) *exec.Cmd {
	gitHubURL := fmt.Sprintf("https://github.com/%s.git", p.Catalog.Repo)

	logrus.Infof("Cloning Rancher-Catalog repo: %s", p.Catalog.Repo)
	// clear if existing and git clone target repo
	return exec.Command("git", "clone", gitHubURL, fmt.Sprintf("%s%s", p.Catalog.Context, repoDir))
}

func addCatalogRepo() *exec.Cmd {
	return exec.Command("git", "add", "-A")
}

func commitCatalogRepo(p Plugin) *exec.Cmd {
	message := fmt.Sprintf("'Update from Drone Build: %d'", p.Build.Number)
	return exec.Command("git", "commit", "-m", message)
}

func pushCatalogRepo() *exec.Cmd {
	return exec.Command("git", "push")
}

// returns true if there are files that need to be commit'd.
func gitChanged() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		logrus.Fatalf("ERROR: Failed to git status %v", err)
	}
	// no output means no changes.
	if len(out) == 0 {
		logrus.Info("No files changed.")
		return false
	}
	logrus.Info("Files changed, add/commit/push changes.")
	return true
}

// this needs to be reworked
func copyIcon(src string, dest string) (*exec.Cmd, error) {
	dir := filepath.Dir(src)
	base := filepath.Base(src)
	// find files in dir that match base
	iconRe := regexp.MustCompile(fmt.Sprintf(`^%s`, base))
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		if iconRe.MatchString(f.Name()) {
			name := fmt.Sprintf("%s/%s", dir, f.Name())
			return exec.Command("cp", name, dest), nil
		}
	}
	return nil, fmt.Errorf("Icon file not found %s", src)
}

// These need work
func parseTemplateFile(file string) (*template.Template, error) {
	name := filepath.Base(file)
	tmpl, err := template.New(name).ParseFiles(file)
	logrus.Debugf("Parsing Template %s", name)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

//
func executeTemplate(target string, tmpl *template.Template, context TemplateContext) error {
	logrus.Debugf("Writing File %s", target)
	targetFile, err := os.Create(target)
	if err != nil {
		return err
	}
	err = tmpl.Execute(targetFile, context)
	if err != nil {
		return err
	}
	targetFile.Close()
	return nil
}

func pickTag(tags []string, tagRegex string) (string, error) {
	var err error
	var tagRe *regexp.Regexp

	logrus.Info("Found the following tags:")
	for _, tag := range tags {
		logrus.Infof(" %s", tag)
	}
	// Use regex first
	if tagRegex != "" {
		tagRe, err = regexp.Compile(tagRegex)
		if err != nil {
			logrus.Infof("Warning: Failed to compile Tag Regex: %s", tagRegex)
		}
	}

	// regex complied lets search the array
	if tagRe != nil {
		for _, tag := range tags {
			if tagRe.MatchString(tag) {
				logrus.Debugf("Using first RegExp matched tag: %s", tag)
				return tag, nil
			}
		}
	}
	// first non-latest tag
	for _, tag := range tags {
		if tag != "latest" {
			logrus.Debugf("Using first tag that is not 'latest' tag: %s", tag)
			return tag, nil
		}
	}
	// latest
	for _, tag := range tags {
		if tag == "latest" {
			logrus.Debug("Using 'latest' tag.")
			return tag, nil
		}
	}
	return "", fmt.Errorf("No valid tags found")
}

func fixName(name string) string {
	name = strings.ToLower(name)
	re := regexp.MustCompile(`[_\.\s]`)
	return re.ReplaceAllString(name, "-")
}

func exists(f string) bool {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return false
	}
	return true
}

func execute(cmd *exec.Cmd, p Plugin) error {
	if p.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// trace only on debug - bleeds credentials
	trace(cmd)

	err := cmd.Run()

	return err
}

// tag so that it can be extracted and displayed in the logs.
// trace writes each command to stdout with the command wrapped in an xml
func trace(cmd *exec.Cmd) {
	logrus.Debugf("%s", strings.Join(cmd.Args, " "))
}
