package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/drone/drone-plugin-go/plugin"
	"github.com/heroku/docker-registry-client/registry"
)

const (
	baseDir                    string = "/rancher-catalog"
	repoDir                    string = "/rancher-catalog/repo"
	templateDir                string = "/rancher-catalog/repo/base"
	dockerComposeTemplateFile  string = "/rancher-catalog/repo/base/docker-compose.tmpl"
	rancherComposeTemplateFile string = "/rancher-catalog/repo/base/rancher-compose.tmpl"
	configTemplateFile         string = "/rancher-catalog/repo/base/config.tmpl"
	iconFileBase               string = "/rancher-catalog/repo/base/catalogIcon"
)

// catalog struct
type catalog struct {
	vargs     vargs
	workspace plugin.Workspace
	repo      plugin.Repo
	build     plugin.Build
}

// vargs strct
type vargs struct {
	DockerRepo     string `json:"docker_repo"`
	DockerUsername string `json:"docker_username"`
	DockerPassword string `json:"docker_password"`
	DockerURL      string `json:"docker_url"`
	CatalogRepo    string `json:"catalog_repo"`
	GitHubToken    string `json:"github_token"`
	GitHubUser     string `json:"github_user"`
	GitHubEmail    string `json:"github_email"`
}

// tagsByBranch struct
type tagsByBranch struct {
	branches map[string]branch
}

// branch struct
type branch struct {
	versions map[string]version
}

// version struct
type version struct {
	builds map[string]*Tag
}

// Tag struct
type Tag struct {
	Tag     string
	Count   int
	Owner   string
	Project string
	Branch  string
	Version string
	Build   string
	SHA     string
}

func main() {
	fmt.Println("starting drone-rancher-catalog...")

	var catalog = catalog{}
	plugin.Param("workspace", &catalog.workspace)
	plugin.Param("repo", &catalog.repo)
	plugin.Param("build", &catalog.build)
	plugin.Param("vargs", &catalog.vargs)
	plugin.MustParse()

	if len(catalog.vargs.DockerRepo) == 0 {
		fmt.Println("ERROR: docker_repo: Docker Registry Repo to read tags from, not specified")
		os.Exit(1)
	}
	if len(catalog.vargs.DockerUsername) == 0 {
		fmt.Println("ERROR: docker_username: Docker Registry Username not specified")
		os.Exit(1)
	}
	if len(catalog.vargs.DockerPassword) == 0 {
		fmt.Println("ERROR: docker_password: Docker Registry Password not specified")
		os.Exit(1)
	}
	if len(catalog.vargs.CatalogRepo) == 0 {
		fmt.Println("ERROR: catalog_repo: GitHub Catalog Repo not specified")
		os.Exit(1)
	}
	if len(catalog.vargs.GitHubToken) == 0 {
		fmt.Println("ERROR: github_token: GitHub User Token not specified")
		os.Exit(1)
	}
	if len(catalog.vargs.DockerURL) == 0 {
		catalog.vargs.DockerURL = "https://registry.hub.docker.com/"
	}
	if len(catalog.vargs.GitHubUser) == 0 {
		catalog.vargs.GitHubUser = catalog.build.Author
	}
	if len(catalog.vargs.GitHubEmail) == 0 {
		catalog.vargs.GitHubEmail = catalog.build.Email
	}

	// create a dir outside the workspace
	if !exists(baseDir) {
		os.Mkdir(baseDir, 0755)
	}

	catalog.cloneCatalogRepo()
	os.Chdir(repoDir)
	catalog.gitConfigureEmail()
	catalog.gitConfigureUser()

	if !exists("./templates") {
		os.Mkdir("./templates", 0755)
	}

	dockerComposeTmpl := catalog.parseTemplateFile(dockerComposeTemplateFile)
	rancherComposeTmpl := catalog.parseTemplateFile(rancherComposeTemplateFile)
	configTmpl := catalog.parseTemplateFile(configTemplateFile)

	tags := catalog.getTags()
	tbb := catalog.tagsByBranch(tags)

	fmt.Println("Creating Catalog Templates for:")
	for branch := range tbb.branches {
		var count int
		var last *Tag

		// create branch dir
		branchDir := fmt.Sprintf("./templates/%s", branch)
		if !exists(branchDir) {
			os.Mkdir(branchDir, 0755)
		}

		// sort versions so we can count builds in a feature branch
		var vKeys []string
		for k := range tbb.branches[branch].versions {
			vKeys = append(vKeys, k)
		}
		sort.Strings(vKeys)

		for _, version := range vKeys {
			// sort builds to count in order
			var bKeys []string
			for k := range tbb.branches[branch].versions[version].builds {
				bKeys = append(bKeys, k)
			}
			sort.Strings(bKeys)

			for _, build := range bKeys {
				tbb.branches[branch].versions[version].builds[build].Count = count

				// create dir structure
				buildDir := fmt.Sprintf("%s/%d", branchDir, count)
				if !exists(buildDir) {
					fmt.Printf("  %d:%s %s-%s\n", count, branch, version, build)
					os.Mkdir(buildDir, 0755)
				}

				// create docker-compose.yml and rancher-compose.yml from template
				// don't generate files if they already exist
				dockerComposeTarget := fmt.Sprintf("%s/docker-compose.yml", buildDir)
				if !exists(dockerComposeTarget) {
					catalog.executeTemplate(dockerComposeTarget, dockerComposeTmpl, tbb.branches[branch].versions[version].builds[build])
				}
				rancherComposeTarget := fmt.Sprintf("%s/rancher-compose.yml", buildDir)
				if !exists(rancherComposeTarget) {
					catalog.executeTemplate(rancherComposeTarget, rancherComposeTmpl, tbb.branches[branch].versions[version].builds[build])
				}

				last = tbb.branches[branch].versions[version].builds[build]
				count++
			}
		}

		// create config.yml from temlplate
		configTarget := fmt.Sprintf("%s/config.yml", branchDir)
		catalog.executeTemplate(configTarget, configTmpl, last)

		// Icon file
		copyIcon(iconFileBase, branchDir)
	}
	// TODO: Delete dir/files if tags don't exist anymore. Need to maintian build dir numbering

	if catalog.gitChanged() {
		catalog.addCatalogRepo()
		catalog.commitCatalogRepo()
		catalog.pushCatalogRepo()
	}
	fmt.Println("... Finished drone-rancher-catalog")
}

func (c *catalog) getTags() []string {
	hub, err := registry.New(c.vargs.DockerURL, c.vargs.DockerUsername, c.vargs.DockerPassword)
	if err != nil {
		fmt.Println("ERROR: Could not Contact Docker Registry", err)
		os.Exit(1)
	}
	tags, err := hub.Tags(c.vargs.DockerRepo)
	if err != nil {
		fmt.Println("ERROR: Getting tags", err)
		os.Exit(1)
	}
	return tags
}

// parseTag Returns a Tag object from a buildgoogles style tag
func (c *catalog) parseTag(t string) *Tag {
	var tag = &Tag{}
	featureRe := regexp.MustCompile(fmt.Sprintf(`^%s_%s_`, c.repo.Owner, c.repo.Name))
	releaseRe := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	// Skip forks and other nonsense tags
	switch {
	case featureRe.MatchString(t):
		// fmt.Println("Found Feature Branch Tag", t)
		tagParts := strings.Split(t, "_")
		// shift the owner and project from the front
		// pop the sha, build, and version from the back
		// join whats left into the branch
		tag.Tag = t
		tag.Owner, tagParts = tagParts[0], tagParts[1:]
		tag.Project, tagParts = tagParts[0], tagParts[1:]
		tag.SHA, tagParts = tagParts[len(tagParts)-1], tagParts[:len(tagParts)-1]
		tag.Build, tagParts = tagParts[len(tagParts)-1], tagParts[:len(tagParts)-1]
		tag.Version, tagParts = tagParts[len(tagParts)-1], tagParts[:len(tagParts)-1]
		tag.Branch = strings.Join(tagParts, "_")
	case releaseRe.MatchString(t):
		// fmt.Println("Found Release Tag", t)
		tag.Tag = t
		tag.Owner = c.repo.Owner
		tag.Project = c.repo.Name
		tag.Branch = "master"
		tag.Build = "1"
		tag.SHA = ""
		versionRe := regexp.MustCompile(`^v`)
		tag.Version = versionRe.ReplaceAllString(t, "")
	default:
		return nil
	}
	return tag
}

// tagsByBranch break down tag list and return a tagsByBranch object
func (c *catalog) tagsByBranch(tags []string) *tagsByBranch {
	tbb := &tagsByBranch{}
	tbb.branches = make(map[string]branch)

	for _, tg := range tags {
		t := c.parseTag(tg)
		if t == nil {
			continue
		}
		if _, present := tbb.branches[t.Branch]; !present {
			tbb.branches[t.Branch] = branch{
				versions: make(map[string]version),
			}
		}
		if _, present := tbb.branches[t.Branch].versions[t.Version]; !present {
			tbb.branches[t.Branch].versions[t.Version] = version{
				builds: make(map[string]*Tag),
			}
		}
		if _, present := tbb.branches[t.Branch].versions[t.Version].builds[t.Build]; !present {
			tbb.branches[t.Branch].versions[t.Version].builds[t.Build] = t
		}
	}
	return tbb
}

func exists(f string) bool {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return false
	}
	return true
}

func (c *catalog) cloneCatalogRepo() {
	gitHubURL := fmt.Sprintf("https://%s:x-oauth-basic@github.com/%s.git", c.vargs.GitHubToken, c.vargs.CatalogRepo)

	fmt.Println("Cloning Rancher-Catalog repo:", c.vargs.CatalogRepo)
	// clear if existing and git clone target repo
	os.RemoveAll(repoDir)

	cmd := exec.Command("git", "clone", gitHubURL, repoDir)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Failed to Clone Repo %v\n", err)
		os.Exit(1)
	}
}

func (c *catalog) addCatalogRepo() {
	cmd := exec.Command("git", "add", "-A")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Failed to git add %v\n", err)
		os.Exit(1)
	}
}

func (c *catalog) commitCatalogRepo() {
	message := fmt.Sprintf("'Update from Drone Build: %d'", c.build.Number)
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Failed to git commit %v\n", err)
		os.Exit(1)
	}
}

func (c *catalog) pushCatalogRepo() {
	cmd := exec.Command("git", "push")
	err := cmd.Run()
	// Not showing output, bleeds the API key
	if err != nil {
		fmt.Printf("ERROR: Failed to git push %v\n", err)
		os.Exit(1)
	}
}

func (c *catalog) parseTemplateFile(file string) *template.Template {
	name := filepath.Base(file)
	tmpl, err := template.New(name).ParseFiles(file)
	if err != nil {
		fmt.Printf("ERROR: Failed parse template %v\n", err)
		os.Exit(1)
	}
	return tmpl
}

func (c *catalog) executeTemplate(target string, tmpl *template.Template, tag *Tag) {
	targetFile, err := os.Create(target)
	if err != nil {
		fmt.Printf("ERROR: Failed to open file %v\n", err)
		os.Exit(1)
	}
	err = tmpl.Execute(targetFile, tag)
	if err != nil {
		fmt.Printf("ERROR: Failed execute template %v\n", err)
		os.Exit(1)
	}
	targetFile.Close()
}

// copy src.* (repo/base/catalogIcon.*) to dest directory
func copy(src string, dest string) {
	cmd := exec.Command("cp", src, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Failed to cp %v\n", err)
		os.Exit(1)
	}
}

func copyIcon(src string, dest string) {
	dir := filepath.Dir(src)
	base := filepath.Base(src)
	// find files in dir that match base
	iconRe := regexp.MustCompile(fmt.Sprintf(`^%s`, base))
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		if iconRe.MatchString(f.Name()) {
			name := fmt.Sprintf("%s/%s", dir, f.Name())
			copy(name, dest)
		}
	}
}

func (c *catalog) gitConfigureEmail() {
	cmd := exec.Command("git", "config", "user.email", c.vargs.GitHubEmail)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Failed to git config %v\n", err)
		os.Exit(1)
	}
}

func (c *catalog) gitConfigureUser() {
	cmd := exec.Command("git", "config", "user.name", c.vargs.GitHubUser)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Failed to git config %v\n", err)
		os.Exit(1)
	}
}

// returns true if there are files that need to be commited.
func (c *catalog) gitChanged() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("ERROR: Failed to git status %v\n", err)
		os.Exit(1)
	}
	// no output means no changes.
	if len(out) == 0 {
		fmt.Println("No files changed.")
		return false
	}
	fmt.Println("Files changed, add/commit/push changes.")
	return true
}
