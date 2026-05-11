package cloud

import "testing"

func TestBuildCloudResourceKeyPrefersCloudIDAndRegion(t *testing.T) {
	networkPlanID := uint(7)
	job := DeploymentJob{
		Provider:      "aws",
		AccountID:     11,
		BlueprintID:   3,
		NetworkPlanID: &networkPlanID,
	}
	entry := ResourceInventoryEntryInput{
		Region:       "ap-southeast-1",
		ResourceType: "aws_subnet",
		ResourceName: "subnet-a",
		CloudID:      "subnet-123456",
	}

	got := buildCloudResourceKey(job, entry)
	want := "aws:11:3:7:ap-southeast-1:aws_subnet:subnet-123456"
	if got != want {
		t.Fatalf("unexpected sync key: got %q want %q", got, want)
	}
}

func TestBuildCloudResourceKeyFallsBackToResourceName(t *testing.T) {
	job := DeploymentJob{
		Provider:    "alicloud",
		AccountID:   5,
		BlueprintID: 2,
	}
	entry := ResourceInventoryEntryInput{
		Region:       "cn-hangzhou",
		ResourceType: "alicloud_vswitch",
		ResourceName: "vsw-private-a",
	}

	got := buildCloudResourceKey(job, entry)
	want := "alicloud:5:2:0:cn-hangzhou:alicloud_vswitch:vsw-private-a"
	if got != want {
		t.Fatalf("unexpected sync key fallback: got %q want %q", got, want)
	}
}
