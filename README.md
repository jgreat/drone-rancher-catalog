# drone-rancher-catalog
Drone CI plugin to publish builds to a custom rancher-catalog.

This plugin will read all of the tags in the specified Docker Hub Repo and create entries based on 2 styles of tags.

* **Release Tags:** `v1.0.1`  
* **Feature Branch Tags:** `jgreat_core-api_feature-branch_1.0.3_3_aaaa1234`

#### Feature Branch Tags Format
`<GithubOwner>_<GithubProject>_<BranchName>_<Version>_<Build>_<SHA>`  
We use [buildgoggles](https://www.npmjs.com/package/buildgoggles) to generate these tags for our nodejs projects and a customized version of [drone-docker](https://hub.docker.com/r/leankit/drone-docker/) to publish programmatic tags to https://hub.docker.com.

#### Catalog Format
This creates a catalog for a single github project. Creates a Entry for each branch with a version of each entry for every build ordered by version/build in each Docker Hub tag.

```bash
base/                        # Go Templates (Create/Edit These)
  |_ catalogIcon.(png|svg)   # Copy for branch/catalogIcon.(png|svg)
  |_ config.tmpl             # Template for branch/config.yml
  |_ rancher-compose.tmpl    # Template for branch/0/rancher-compose.yml
  |_ docker-compose.tmpl     # Template for branch/0/docker-compose.yml

templates/                   # Generated Catalog (don't manually edit)
  |_ master/                 # based on branch name (master for Release Tag)
    |_ config.yml            # Catalog Entry config
    |_ catalogIcon.png       # icon image
    |_ 0/                    # builds for a branch
      |_ docker-compose.yml  # Entry/Build docker-compose.yml
      |_ rancher-compose.yml # Entry/Build rancher-compose.yml
    |_ 1/
      |_ docker-compose.yml
      |_ rancher-compose.yml
  |_ feature-branch/
    |_ config.yml
    |_ catalogIcon.png
    |_ 0/
      |_ docker-compose.yml
      |_ rancher-compose.yml
  ...
```

### Usage
#### Create a Catalog Repo
Create a GitHub Repo.

[Sample Repo to get you started.](https://github.com/jgreat/drone-rancher-catalog-base)

#### Populate templates
drone-rancher-catalog will read the templates from the `base/` directory and build the Catalog `templates/` directory.

`*.tmpl` files are [Go templates](https://golang.org/pkg/text/template/)

Templates are executed with:
```go
{{ .Tag }}     // Full tag from Docker Hub
{{ .Count }}   // Which Build in Branch
{{ .Owner }}   // Github Owner of Docker Tag or Drone Build
{{ .Project }} // Github Repo of Docker Tag or Drone Build
{{ .Branch }}  // Branch of Docker Tag or master for Release
{{ .Version }} // Version of Docker Tag
{{ .Build }}   // Build of Docker Tag or 1 for Release
{{ .SHA }}     // Short SHA of Docker Tag or "" for Release
```

#### .drone.yml
Add a publish step to your project .drone.yml.

This will require you to add `jgreat/drone-rancher-catalog` to the `PLUGIN_FILTER` variable when you run the drone server.

```yaml
publish:
  rancher-catalog:
    image: jgreat/docker-rancher-catalog
    docker_username: $$DOCKER_USER        # Docker Hub Username
    docker_password: $$DOCKER_PASS        # Docker Hub Password
    docker_repo: jgreat/core-api          # Docker Hub Repo
    catalog_repo: jgreat/rancher-core-api # Rancher Catalog to Populate
    github_token: $$GITHUB_TOKEN          # Personal API Token
    github_user: Jason Greathouse         # GitHub Username (for git commit)
    github_email: jason@jgreat.me         # GitHub Email (for git commit)
```

### TODO
* Idea: Some kind of tag matching template and regex to allow any kind of tags.
* Issue: Don't delete tags from your registry. Will mess up the catalog dir order. Need to create and maintain a simple index of builds.

### Build
build with go version 1.6

Restore Dependencies
```
godep restore
```

Build
```
build.sh
```

Test Run
```
run.sh
```
