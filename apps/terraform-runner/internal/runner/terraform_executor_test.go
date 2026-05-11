package runner

import (
	"strings"
	"testing"
)

func TestRedactSensitiveText(t *testing.T) {
	input := strings.Join([]string{
		"AWS_SECRET_ACCESS_KEY=super-secret",
		`{"secret_key":"abc123","token":"jwt-value","private_key":"pem-data"}`,
	}, "\n")

	got := redactSensitiveText(input)

	for _, needle := range []string{"super-secret", "abc123", "jwt-value", "pem-data"} {
		if strings.Contains(got, needle) {
			t.Fatalf("expected sensitive value %q to be redacted, got %q", needle, got)
		}
	}
	if !strings.Contains(got, "***REDACTED***") {
		t.Fatalf("expected redaction marker in output, got %q", got)
	}
}

func TestCollectStateResourcesIncludesCloudIDAndMetadata(t *testing.T) {
	payload := map[string]any{
		"root_module": map[string]any{
			"resources": []any{
				map[string]any{
					"address":       "aws_instance.bastion",
					"mode":          "managed",
					"type":          "aws_instance",
					"name":          "bastion",
					"provider_name": "registry.terraform.io/hashicorp/aws",
					"values": map[string]any{
						"id":                "i-1234567890",
						"arn":               "arn:aws:ec2:ap-southeast-1:123456789012:instance/i-1234567890",
						"public_ip":         "54.1.1.1",
						"private_ip":        "10.60.0.10",
						"availability_zone": "ap-southeast-1a",
						"instance_type":     "t3.micro",
						"tags": map[string]any{
							"Name": "bastion-demo",
							"Role": "bastion",
						},
					},
				},
			},
		},
	}

	items := collectStateResources(payload)
	if len(items) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(items))
	}
	resource := items[0]
	if got := resource["cloud_id"]; got != "i-1234567890" {
		t.Fatalf("expected cloud_id to use state id, got %#v", got)
	}
	metadata, ok := resource["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata map, got %#v", resource["metadata"])
	}
	for key, want := range map[string]any{
		"id":                "i-1234567890",
		"arn":               "arn:aws:ec2:ap-southeast-1:123456789012:instance/i-1234567890",
		"public_ip":         "54.1.1.1",
		"private_ip":        "10.60.0.10",
		"availability_zone": "ap-southeast-1a",
		"instance_type":     "t3.micro",
	} {
		if got := metadata[key]; got != want {
			t.Fatalf("expected metadata[%q]=%#v, got %#v", key, want, got)
		}
	}
	tags, ok := metadata["tags"].(map[string]any)
	if !ok || tags["Name"] != "bastion-demo" {
		t.Fatalf("expected tags metadata, got %#v", metadata["tags"])
	}
}
