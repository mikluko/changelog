package main

import (
	"path"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"sarak.sh/mikluko/changelog/v2/pkg/changelog"
)

// The release flow under skills/changelog/references/workflows is copied into
// a repository's .github/workflows as a set. A rename in one file leaves
// release.yaml calling a workflow, passing an input or downloading an artifact
// the set no longer has, and nothing but these tests reads the four together.
const (
	flowDir     = "skills/changelog/references/workflows"
	flowRelease = flowDir + "/release.yaml"
	flowBuild   = flowDir + "/build.yaml"
	flowLint    = flowDir + "/lint.yaml"

	// flowCall is how a workflow names a reusable one of its own repository.
	flowCall = "./.github/workflows/"
)

// flowWorkflow is the part of a workflow the flow's files are held together by.
type flowWorkflow struct {
	On map[string]struct {
		Inputs map[string]yaml.Node `yaml:"inputs"`
	} `yaml:"on"`
	Jobs map[string]struct {
		Uses  string            `yaml:"uses"`
		With  map[string]string `yaml:"with"`
		Steps []struct {
			Uses string            `yaml:"uses"`
			With map[string]string `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func readFlow(t *testing.T, name string) flowWorkflow {
	t.Helper()
	var w flowWorkflow
	if err := yaml.Unmarshal(read(t, name), &w); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return w
}

// artifacts returns the artifact names the steps of the workflow at name pass
// to the action whose reference starts with prefix.
func artifacts(t *testing.T, name, prefix string) []string {
	t.Helper()
	var out []string
	for _, job := range readFlow(t, name).Jobs {
		for _, step := range job.Steps {
			if strings.HasPrefix(step.Uses, prefix) {
				out = append(out, step.With["name"])
			}
		}
	}
	return out
}

// Every workflow release.yaml calls ships beside it, can be called, and
// declares each input it is passed.
func TestReleaseFlowCallsTheWorkflowsBesideIt(t *testing.T) {
	calls := 0
	for name, job := range readFlow(t, flowRelease).Jobs {
		file, ok := strings.CutPrefix(job.Uses, flowCall)
		if !ok {
			continue
		}
		calls++
		call, ok := readFlow(t, path.Join(flowDir, file)).On["workflow_call"]
		if !ok {
			t.Errorf("job %s calls %s, which carries no workflow_call trigger", name, file)
		}
		for input := range job.With {
			if _, ok := call.Inputs[input]; !ok {
				t.Errorf("job %s passes %s the input %q, which it does not declare", name, file, input)
			}
		}
	}
	if calls == 0 {
		t.Fatalf("%s calls no workflow; this test is checking nothing", flowRelease)
	}
}

// The image crosses from build.yaml to release.yaml as a workflow artifact,
// named once in each file.
func TestReleaseFlowPushesTheArtifactTheBuildLeaves(t *testing.T) {
	up := artifacts(t, flowBuild, "actions/upload-artifact@")
	down := artifacts(t, flowRelease, "actions/download-artifact@")
	if len(up) != 1 || up[0] == "" || !reflect.DeepEqual(up, down) {
		t.Errorf("build.yaml uploads %q and release.yaml downloads %q", up, down)
	}
}

// The skill says notes live in CHANGELOG.md under this flow, and that main is
// held to one check more than a pull request: both invocations switch
// fragments off, and prerelease-entry is the only input they differ in.
func TestReleaseFlowInvocationsDifferInOneInput(t *testing.T) {
	on, off := exampleInputs(t, flowRelease), exampleInputs(t, flowLint)

	if got, ok := off["fragments"]; !ok || got != "" {
		t.Errorf("the pull-request invocation sets fragments to %q, set %t; it is meant to check none", got, ok)
	}
	if got := on["error"]; got != changelog.CheckPrereleaseEntry {
		t.Errorf("the main-branch invocation raises %q, not %s", got, changelog.CheckPrereleaseEntry)
	}

	delete(on, "error")
	if !reflect.DeepEqual(on, off) {
		t.Errorf("the two invocations differ in more than prerelease-entry:\n%v\n%v", on, off)
	}
}
