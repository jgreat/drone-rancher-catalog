package main

import (
	"fmt"
	"testing"

	"github.com/Sirupsen/logrus"
)

type pickTagsTestSet struct {
	description string
	tags        []string
	tagRegex    string
	result      string
	err         error
}

var pickTagTests = []pickTagsTestSet{
	{
		description: "No regex - Should return first non-latest tag",
		tags:        []string{"1.0.1-1499290301.test-branch.1234abcd", "1.0.1", "latest"},
		tagRegex:    "",
		result:      "1.0.1-1499290301.test-branch.1234abcd",
		err:         nil,
	},
	{
		description: "Use semver regex - Should return regex match",
		tags:        []string{"1.0.1-1499290301.test-branch.1234abcd", "1.0.1", "latest"},
		tagRegex:    "[0-9]+[.][0-9]+[.][0-9]+$",
		result:      "1.0.1",
		err:         nil,
	},
	{
		description: "Latest - Should return latest tag",
		tags:        []string{"latest"},
		tagRegex:    "[0-9]+[.][0-9]+[.][0-9]+$",
		result:      "latest",
		err:         nil,
	},
	{
		description: "No latest and not regex match - Should return first non latest (only) tag",
		tags:        []string{"1.0.1-1499290301.test-branch.1234abcd"},
		tagRegex:    "[0-9]+[.][0-9]+[.][0-9]+$",
		result:      "1.0.1-1499290301.test-branch.1234abcd",
		err:         nil,
	},
	{
		description: "No tags provided - Should return blank and error",
		tags:        []string{},
		tagRegex:    "[0-9]+[.][0-9]+[.][0-9]+$",
		result:      "",
		err:         fmt.Errorf("No valid tags found"),
	},
}

func TestPickTags(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)
	for _, set := range pickTagTests {
		t.Log(set.description)
		result, err := pickTag(set.tags, set.tagRegex)
		if result != set.result {
			t.Error(
				"For tags", set.tags,
				"\nFor tagRegex", set.tagRegex,
				"\nexpected result", set.result,
				"\nexpected err", set.err,
				"\ngot result", result,
				"\ngot err", err,
			)
		}
	}
}

type fixNameTestSet struct {
	description string
	name        string
	result      string
}

var fixNameTests = []fixNameTestSet{
	{
		description: "Should replace underscores, dots and spaces with a dash",
		name:        "this_is a weird.name",
		result:      "this-is-a-weird-name",
	},
	{
		description: "Should replace UPPERCASE with lowercase",
		name:        "thisIsAWeirdName2Have",
		result:      "thisisaweirdname2have",
	},
}

func TestFixName(t *testing.T) {
	for _, set := range fixNameTests {
		t.Log(set.description)
		result := fixName(set.name)
		if result != set.result {
			t.Error(
				"For name", set.name,
				"\nexpected result", set.result,
				"\ngot result", result,
			)
		}
	}
}
