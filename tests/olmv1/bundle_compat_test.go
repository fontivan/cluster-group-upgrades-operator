package olmv1_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type csvDocument struct {
	Spec struct {
		InstallModes []struct {
			Type      string `yaml:"type"`
			Supported bool   `yaml:"supported"`
		} `yaml:"installModes"`
		WebhookDefinitions any `yaml:"webhookdefinitions"`
	} `yaml:"spec"`
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "bundle", "metadata", "annotations.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

func TestBundleOLMv1InstallModes(t *testing.T) {
	root := repoRoot(t)
	csvPath := filepath.Join(root, "bundle", "manifests", "cluster-group-upgrades-operator.clusterserviceversion.yaml")
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}

	var csv csvDocument
	if err := yaml.Unmarshal(data, &csv); err != nil {
		t.Fatal(err)
	}

	modes := map[string]bool{}
	for _, mode := range csv.Spec.InstallModes {
		modes[mode.Type] = mode.Supported
	}

	if !modes["AllNamespaces"] {
		t.Error("AllNamespaces must be supported for OLM v1")
	}
	for _, unsupported := range []string{"OwnNamespace", "SingleNamespace", "MultiNamespace"} {
		if modes[unsupported] {
			t.Errorf("%s must not be supported", unsupported)
		}
	}
	if csv.Spec.WebhookDefinitions != nil {
		t.Error("CSV must not declare webhookdefinitions for OLM v1 compatibility tests")
	}
}

func TestBundleRegistryV1MediaType(t *testing.T) {
	root := repoRoot(t)
	annotationsPath := filepath.Join(root, "bundle", "metadata", "annotations.yaml")
	data, err := os.ReadFile(annotationsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "operators.operatorframework.io.bundle.mediatype.v1: registry+v1") {
		t.Fatal("bundle must use registry+v1 format")
	}
}

func TestDeployOLMv1ReferenceCRsPresent(t *testing.T) {
	root := repoRoot(t)
	required := []string{
		"namespace.yaml",
		"serviceaccount.yaml",
		"clusterextension.yaml",
		"clustercatalog.yaml",
		"kustomization.yaml",
	}
	for _, file := range required {
		path := filepath.Join(root, "deploy", "olmv1", file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing reference CR %s: %v", file, err)
		}
	}

	ce, err := os.ReadFile(filepath.Join(root, "deploy", "olmv1", "clusterextension.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(ce)
	if !strings.Contains(content, "kind: ClusterExtension") {
		t.Fatal("clusterextension.yaml must define ClusterExtension")
	}
	if !strings.Contains(content, "packageName: topology-aware-lifecycle-manager") {
		t.Fatal("clusterextension.yaml must reference topology-aware-lifecycle-manager package")
	}
	if strings.Contains(content, "watchNamespace:") {
		t.Fatal("AllNamespaces-only bundle must not set watchNamespace in ClusterExtension spec")
	}
}
