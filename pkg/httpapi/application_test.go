package httpapi

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/redhat-developer/gitops-backend/pkg/parser"
)

const (
	partOfLabel = "app.kubernetes.io/part-of"
)

const testSourceURL = "https://github.com/rhd-example-gitops/gitops-demo.git"

func TestParseServicesFromResources(t *testing.T) {
	goDemoResources := []*parser.Resource{
		{
			Group: "apps", Version: "v1", Kind: "Deployment", Name: "go-demo-http",
			Labels: map[string]string{
				nameLabel:   "go-demo",
				partOfLabel: "go-demo",
			},
			Images: []string{"bigkevmcd/go-demo:876ecb3"},
		},
		{
			Version: "v1", Kind: "Service", Name: "go-demo-http",
			Labels: map[string]string{
				nameLabel:   "go-demo",
				partOfLabel: "go-demo",
			},
		},
		{
			Version: "v1", Kind: "ConfigMap", Name: "go-demo-config",
			Labels: map[string]string{
				nameLabel:   "go-demo",
				partOfLabel: "go-demo",
			},
		},
	}

	redisResources := []*parser.Resource{
		{
			Version: "v1", Kind: "Service", Name: "redis",
			Labels: map[string]string{
				nameLabel:   "redis",
				partOfLabel: "go-demo",
			},
		},
		{
			Group: "apps", Version: "v1", Kind: "Deployment", Name: "redis",
			Labels: map[string]string{
				nameLabel:   "redis",
				partOfLabel: "go-demo",
			},
			Images: []string{"redis:6-alpine"},
		},
	}
	env := &environment{
		Name:    "test-env",
		Cluster: "https://cluster.local",
		Apps: []*application{
			{
				Name: "my-app",
				Services: []service{
					{
						Name:      "go-demo",
						SourceURL: testSourceURL,
					},
					{
						Name: "redis",
					},
				},
			},
		},
	}
	res := append(goDemoResources, redisResources...)

	svcs, err := parseServicesFromResources(env, res)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].Name < svcs[j].Name
	})

	want := []responseService{
		{
			Name: "go-demo",
			Source: source{
				URL:  testSourceURL,
				Type: "github.com",
			},
			Images:    []string{"bigkevmcd/go-demo:876ecb3"},
			Resources: goDemoResources,
		},
		{
			Name:      "redis",
			Source:    source{},
			Images:    []string{"redis:6-alpine"},
			Resources: redisResources,
		},
	}
	if diff := cmp.Diff(want, svcs); diff != "" {
		t.Fatalf("parseServicesFromResources got\n%s", diff)
	}
}

func TestParseServicesFromResourcesReturnsSetOfImages(t *testing.T) {
	res := []*parser.Resource{
		{
			Group: "apps", Version: "v1", Kind: "Deployment", Name: "go-demo-http",
			Labels: map[string]string{
				nameLabel:   "go-demo",
				partOfLabel: "go-demo",
			},
			Images: []string{"bigkevmcd/go-demo:876ecb3"},
		},
		{
			Group: "apps", Version: "v1", Kind: "Deployment", Name: "go-demo-cmd",
			Labels: map[string]string{
				nameLabel:   "go-demo",
				partOfLabel: "go-demo",
			},
			Images: []string{"bigkevmcd/testing:a29fcef", "bigkevmcd/go-demo:876ecb3"},
		},
	}
	env := &environment{
		Name:    "test-env",
		Cluster: "https://cluster.local",
		Apps: []*application{
			{
				Name: "my-app",
				Services: []service{
					{
						Name:      "go-demo",
						SourceURL: testSourceURL,
					},
				},
			},
		},
	}

	svcs, err := parseServicesFromResources(env, res)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].Name < svcs[j].Name
	})

	want := []responseService{
		{
			Name: "go-demo",
			Source: source{
				URL:  testSourceURL,
				Type: "github.com",
			},
			Images: []string{
				"bigkevmcd/go-demo:876ecb3",
				"bigkevmcd/testing:a29fcef",
			},
			Resources: res,
		},
	}
	if diff := cmp.Diff(want, svcs); diff != "" {
		t.Fatalf("parseServicesFromResources got\n%s", diff)
	}
}

func TestParseServicesFromResourcesIgnoresEmptyServices(t *testing.T) {
	res := []*parser.Resource{
		{
			Group: "apps", Version: "v1", Kind: "Deployment", Name: "go-demo-http",
			Labels: map[string]string{},
			Images: []string{"bigkevmcd/go-demo:876ecb3"},
		},
		{
			Version: "v1", Kind: "Service", Name: "go-demo-http",
			Labels: map[string]string{},
		},
	}
	env := &environment{
		Name:    "test-env",
		Cluster: "https://cluster.local",
		Apps: []*application{
			{
				Name: "my-app",
				Services: []service{
					{
						Name:      "go-demo",
						SourceURL: testSourceURL,
					},
				},
			},
		},
	}

	svcs, err := parseServicesFromResources(env, res)
	if err != nil {
		t.Fatal(err)
	}

	want := []responseService{}
	if diff := cmp.Diff(want, svcs); diff != "" {
		t.Fatalf("parseServicesFromResources got\n%s", diff)
	}
}

func TestParseServicesFromResourcesIgnoresUnknownServices(t *testing.T) {
	res := []*parser.Resource{
		{
			Group: "apps", Version: "v1", Kind: "Deployment", Name: "go-demo-http",
			Labels: map[string]string{
				nameLabel:   "unknown",
				partOfLabel: "unknown",
			},
			Images: []string{"bigkevmcd/go-demo:876ecb3"},
		},
	}

	env := &environment{
		Name:    "test-env",
		Cluster: "https://cluster.local",
		Apps: []*application{
			{
				Name: "my-app",
				Services: []service{
					{
						Name:      "go-demo",
						SourceURL: testSourceURL,
					},
				},
			},
		},
	}

	svcs, err := parseServicesFromResources(env, res)
	if err != nil {
		t.Fatal(err)
	}

	want := []responseService{}
	if diff := cmp.Diff(want, svcs); diff != "" {
		t.Fatalf("parseServicesFromResources got\n%s", diff)
	}
}
