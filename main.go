package main

// drone-rancher-catalog
// Build rancher catalog entries for completed builds
// Plugin layout based off of github.com/drone-plugins/drone-docker

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var version = "1.0.1"

func main() {
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		godotenv.Overload(env)
	}

	app := cli.NewApp()
	app.Name = "drone-rancher-catalog plugin"
	app.Usage = "drone-rancher-catalog plugin"
	app.Action = run
	app.Version = version
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "dry_run",
			Usage:  "dry run disables github commit and push",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "Debug output",
			EnvVar: "PLUGIN_DEBUG",
		},
		// Build Flags
		cli.StringFlag{
			Name:   "context",
			Usage:  "build context",
			Value:  ".",
			EnvVar: "PLUGIN_CONTEXT",
		},
		cli.StringFlag{
			Name:   "catalog_context",
			Usage:  "catalog directory context",
			Value:  "/",
			EnvVar: "PLUGIN_CATALOG_CONTEXT",
		},
		cli.StringFlag{
			Name:   "release_branch",
			Usage:  "Release branch to rename to match project",
			EnvVar: "PLUGIN_RELEASE_BRANCH",
		},
		cli.StringSliceFlag{
			Name:   "tags",
			Usage:  "tags",
			EnvVar: "PLUGIN_TAG,PLUGIN_TAGS",
		},
		cli.StringFlag{
			Name:   "tag_regex",
			Usage:  "Regex to pick your tag.",
			EnvVar: "PLUGIN_TAG_REGEX",
			Value:  "",
		},
		cli.StringFlag{
			Name:   "build_commit_branch",
			Usage:  "git commit branch",
			EnvVar: "DRONE_COMMIT_BRANCH",
		},
		cli.StringFlag{
			Name:   "build_repo_name",
			Usage:  "build github repo name",
			EnvVar: "DRONE_REPO_NAME",
		},
		cli.StringFlag{
			Name:   "build_number",
			Usage:  "drone build number",
			EnvVar: "DRONE_BUILD_NUMBER",
		},
		// Github
		cli.StringFlag{
			Name:   "github_email",
			Usage:  "github email",
			EnvVar: "PLUGIN_GITHUB_EMAIL,GITHUB_EMAIL",
		},
		cli.StringFlag{
			Name:   "github_username",
			Usage:  "github username",
			EnvVar: "PLUGIN_GITHUB_USERNAME,GITHUB_USERNAME",
		},
		cli.StringFlag{
			Name:   "github_token",
			Usage:  "github api token",
			EnvVar: "PLUGIN_GITHUB_TOKEN,GITHUB_TOKEN",
		},
		// Rancher catalog - this is a github 'owner/repo'
		cli.StringFlag{
			Name:   "catalog_repo",
			Usage:  "github catalog repo",
			EnvVar: "PLUGIN_CATALOG_REPO",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	plugin := Plugin{
		Dryrun:   c.Bool("dry_run"),
		Debug:    c.Bool("debug"),
		TagRegex: c.String("tag_regex"),
		Catalog: Catalog{
			Context:       c.String("catalog_context"),
			Repo:          c.String("catalog_repo"),
			ReleaseBranch: c.String("release_branch"),
		},
		Github: Github{
			Username: c.String("github_username"), // can we just use a an api token here?
			Token:    c.String("github_token"),    // ^^^
			Email:    c.String("github_email"),
		},
		Build: Build{
			Repo:    c.String("build_repo_name"),
			Number:  c.Int64("build_number"),
			Branch:  c.String("build_commit_branch"),
			Tags:    c.StringSlice("tags"),
			Context: c.String("build_context"),
		},
	}

	// there has to be a better way to check flags
	required := []string{
		"catalog_repo",
		"github_username",
		"github_token",
		"github_email",
		"build_repo_name",
		"build_number",
		"build_commit_branch",
		"tags",
	}

	// cli package doesn't seem have a way to return the Usage for a flag :(
	for _, flag := range required {
		present := c.IsSet(flag)
		if !present {
			logrus.Fatalf("Missing Required Flag %s", flag)
		}
	}

	return plugin.Exec()
}
