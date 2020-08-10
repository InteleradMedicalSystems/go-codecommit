package codecommit

import (
	"testing"
)

func TestCloneURLRegion(t *testing.T) {
	e := NewCloneURLTest(t,
		TestOptions{
			uRL:    "https://git-codecommit.ca-central-1.amazonaws.com/v1/repos/ops-cloudpacs-dev",
			region: "ca-central-1",
			method: "get",
		})
	e.assertRegion()
}
func TestCloneURLInvalidRegion(t *testing.T) {
	e := NewCloneURLTest(t,
		TestOptions{
			uRL:    "",
			region: "ca-central-1",
			method: "get",
		})
	e.assertInvalidRegion()
}

func NewCloneURLTest(t *testing.T, topts TestOptions) *CloneURLTest {
	return &CloneURLTest{
		t,
		topts,
	}
}

type TestOptions struct {
	uRL    string
	region string
	method string
}

type CloneURLTest struct {
	t     *testing.T
	topts TestOptions
}

func (e *CloneURLTest) assertInvalidRegion() {
	t := e.t
	c, err := NewCloneURL(nil, e.topts.uRL)
	if err != nil {
		t.Error(err)
	}

	actual, err := c.parseRegion()
	if err == nil {
		t.Errorf("expected error not returned, actual %q", actual)
	}
}

func (e *CloneURLTest) assertRegion() {
	t := e.t
	c, err := NewCloneURL(nil, e.topts.uRL)
	if err != nil {
		t.Error(err)
	}
	if err != nil {
		t.Error(err)
	}

	actual, err := c.parseRegion()
	if err != nil {
		t.Error(err)
	}
	expected := e.topts.region
	if actual != expected {
		t.Errorf("expected region %q, actual %q", expected, actual)
	}
}
