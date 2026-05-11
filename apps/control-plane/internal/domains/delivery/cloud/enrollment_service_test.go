package cloud

import "testing"

func TestStringValueIgnoresNilSentinel(t *testing.T) {
	if got := stringValue(nil); got != "" {
		t.Fatalf("expected empty string for nil, got %q", got)
	}
	if got := stringValue("<nil>"); got != "" {
		t.Fatalf("expected empty string for nil sentinel, got %q", got)
	}
	if got := stringValue(" otc "); got != "otc" {
		t.Fatalf("expected trimmed string, got %q", got)
	}
}

func TestResolveClusterEnrollmentResourceIDPrefersCloudID(t *testing.T) {
	item := CloudResource{CloudID: "cluster-a"}
	input := ClusterEnrollmentResultInput{
		Metadata: map[string]any{
			"cloud_id": "cluster-b",
		},
	}
	cloudMetadata := map[string]any{
		"cloud_id": "cluster-c",
		"metadata": map[string]any{
			"cloud_id": "cluster-d",
		},
	}
	if got := resolveClusterEnrollmentResourceID(item, input, cloudMetadata); got != "cluster-a" {
		t.Fatalf("expected item cloud id to win, got %q", got)
	}
}

func TestResolveClusterEnrollmentNameFallsBackToMetadataName(t *testing.T) {
	item := CloudResource{ResourceName: "resource-name"}
	cloudMetadata := map[string]any{
		"metadata": map[string]any{
			"name": "cluster-name",
		},
	}
	if got := resolveClusterEnrollmentName(item, ClusterEnrollmentResultInput{}, cloudMetadata, "cluster-id"); got != "cluster-name" {
		t.Fatalf("expected metadata name fallback, got %q", got)
	}
}
