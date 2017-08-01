# drone-rancher-catalog

Drone CI plugin to publish builds to a custom rancher-catalog.

This plugin has been simplified for drone 0.7+ (may work with 0.5+)

### Usage

#### Create a Catalog Repo

Create a GitHub Repo. This is `catalog_repo:` value in ``.drone.yml`

[Sample Repo to get you started.](https://github.com/jgreat/drone-rancher-catalog-base)

#### Catalog Format

This creates a catalog for a single github project. Creates a Entry for each branch with a version of each entry for every build ordered by version/build.

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
    |_ 0/                    # Drone Build Number
      |_ docker-compose.yml  # Entry/Build docker-compose.yml
      |_ rancher-compose.yml # Entry/Build rancher-compose.yml
    |_ 1/
      |_ docker-compose.yml
      |_ rancher-compose.yml
  |_ feature-branch/
    |_ config.yml
    |_ catalogIcon.png
    |_ 5/
      |_ docker-compose.yml
      |_ rancher-compose.yml
  ...
```

#### Populate templates

drone-rancher-catalog will read the templates from the `base/` directory and build the Catalog `templates/` directory.

`*.tmpl` files are [Go templates](https://golang.org/pkg/text/template/)

Templates are executed with:

```go
{{ .Build }}      // Drone Build Number
{{ .Branch }}     // Github Branch from Drone Build
{{ .Project }}    // Github Repo from Drone Build normalized for Rancher (lowercase and replace [_\.\s] with -)
{{ .Tag }}        // Docker Image Tag
```

#### .drone.yml

Add a step in your pipeline to your project `.drone.yml`.

```yaml
pipeline:
  generate_rancher_catalog:
    image: jgreat/drone-rancher-catalog
    env_file: .env                            # env file that will set environment variables for the build.
    catalog_repo: jgreat/rancher-test-catalog # Rancher Catalog Github Repo
    tag_regex: "[0-9]+[.][0-9]+[.][0-9]+"     # See Tag RegEx
    release_branch: master                    # See Standard Options
    tags:                                     # Docker Image Tags (See Tags for more info)
      - latest
      - 1.0.1
      - 1.0.1-23.test-branch.1234abcd
    secrets:
      - github_email                          # Github Credentials
      - github_username
      - github_token
    when:
      event: [ push ]                         # Normally we only want to publish on a push

```

### Options

#### Standard Options

* `catalog_repo`: Github Repo for your Rancher Catalog. See [Catalog Format](#catalog-format).
* `tags`: List of image tags. See [Tags](#tags) for more advanced options.
* `tag_regex`: (Optional) Regular Expression to choose what docker image tag to use for the catalog. See [Tags](#tags) for more advanced options.
* `release_branch`: (Optional) Branch name to use as the release. The `{{ .Branch }}` template parameter will be set = to `{{ .Project }}` parameter. The catalog entry will be named with the `{{ .Project }}` parameter.
* `env_file`: (Optional) Path to an env file. This can be used to define/override environment variables. See [Tags](#tags) and [Build Numbers](#build-numbers) for some uses.

#### Secrets

Define your github credentials as Drone secrets.

* `github_email`: Email address for commit.
* `github_username`: Username for commit.
* `github_token`: Personal access token for Rancher Catalog Repo.

### Tags

Since we set the Rancher Catalog `version` to the docker image tag, the tags you use for the image must be valid [SemVer](http://semver.org/).

#### Tag Order

The order you list the tags matters. Only one image tag is going to be specified in the catalog entry.

1. `tag_regex` match
1. First tag not `latest`

If no valid tags are specified the catalog creation will error out.

#### Tag RegEx

Use the `tag_regex` option to pick the tag pattern you want to use when you build your docker image with multiple tags. Since escaping is hard, I suggest using bracketed character sets instead of \ shortcuts.

For example we push our docker images with 2 tags.

* 0.1.1  (Release tag)
* 0.1.1-123456.master.1234abcd (Detailed tag `version`-`buildnumber`.`branch`.`sha`)

We want to use the shorter release tag.

```yaml
tag_regex: "[0-9]+[.][0-9]+[.][0-9]$"
```

#### Static Tags

You can specify a list of tags in your .drone.yml in the same format as the docker plugin.

#### Programmatic Tags

Programmatic tags can be set as part of the build. You can pass the list of tags on to the docker plugin and this plugin in an env file.  

Set the path to the env file with the `env_file` file key.

```yaml
pipeline:
  generate_rancher_catalog:
    image: jgreat/drone-rancher-catalog
    env_file: .env
```

Set the contents of the env file in a previous build step. `PLUGIN_TAGS` is a comma separated list of tags.

```bash
PLUGIN_TAGS=1.0.1-129840.feature-branch.abcd1234,1.0.1,latest
```

### Build Numbers

Build numbers must be an integer. If you don't want to use Drone's build number, you can override it by using a .env file

```bash
DRONE_BUILD_NUMBER=129840
```

## Build drone-rancher-catalog

build with go version 1.8

### Install Glide

```bash
curl https://glide.sh/get | sh
```

### Test and Build

This installs the dependencies, runs tests, builds the executable, builds a docker images.

```bash
./build.sh
```