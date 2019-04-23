package main

import (
	"testing"
)

/* TODO: write tests.
func TestRenderBasic(t *testing.T) {
	e := NewCodeCommitCredentialsTest(t,
		TestOptions{
			uRL:    "https://git-codecommit.ca-central-1.amazonaws.com/v1/repos/ops-cloudpacs-dev",
			region: "ca-central-1",
			method: "get",
		})
}
*/

func NewCodeCommitCredentialsTest(t *testing.T, topts TestOptions) *CodeCommitCredentialsTest {
	return &CodeCommitCredentialsTest{
		t,
		topts,
	}
}

type TestOptions struct {
	uRL    string
	region string
	method string
}

type CodeCommitCredentialsTest struct {
	t     *testing.T
	topts TestOptions
}
