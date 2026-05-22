package cloud

import (
	"strings"
	"testing"
	"time"
)

func TestBuildCloudWorkbenchStartsWithAccountSetup(t *testing.T) {
	view := buildCloudWorkbench(cloudWorkbenchData{
		Summary:  Summary{Accounts: 1},
		Accounts: []CloudAccountView{{ID: 1, Name: "aws-prod", Status: "failed"}},
	})

	assertWorkbenchIdentity(t, view)
	assertWorkbenchStage(t, view, "account", "create-account")
	assertWorkbenchBlocker(t, view, "没有可用云账号")
}

func TestBuildCloudWorkbenchMovesToFoundationNetworkAfterAccountReady(t *testing.T) {
	view := buildCloudWorkbench(cloudWorkbenchData{
		Summary:  Summary{Accounts: 1},
		Accounts: []CloudAccountView{{ID: 1, Name: "aws-prod", Status: "active"}},
	})

	assertWorkbenchStage(t, view, "foundation_network", "create-network")
	assertWorkbenchBlocker(t, view, "还没有基础网络")
}

func TestBuildCloudWorkbenchRequiresSuccessfulFoundationApply(t *testing.T) {
	view := buildCloudWorkbench(cloudWorkbenchData{
		Summary:      Summary{Accounts: 1, Networks: 1, Jobs: 1},
		Accounts:     []CloudAccountView{{ID: 1, Name: "aws-prod", Status: "active"}},
		NetworkPlans: []NetworkPlanView{{ID: 7, Name: "prod-base", Status: "planned"}},
		Blueprints:   []BlueprintView{{ID: 11, Code: "aws-vpc-base", Name: "AWS 基础网络", Category: "network"}},
		Jobs: []DeploymentJobView{{
			ID:            101,
			Name:          "prod-base-plan",
			BlueprintID:   11,
			BlueprintName: "AWS 基础网络",
			Status:        "planned",
			Action:        "apply",
			CreatedAt:     time.Now(),
		}},
	})

	assertWorkbenchStage(t, view, "foundation_apply", "scroll-networks")
	assertWorkbenchBlocker(t, view, "还没有成功的基础交付任务")
}

func TestBuildCloudWorkbenchGuidesDeliveryObjectsAfterFoundationReady(t *testing.T) {
	view := buildCloudWorkbench(cloudWorkbenchData{
		Summary:      Summary{Accounts: 1, Networks: 1, Jobs: 1, Resources: 6},
		Accounts:     []CloudAccountView{{ID: 1, Name: "aws-prod", Status: "active"}},
		NetworkPlans: []NetworkPlanView{{ID: 7, Name: "prod-base", Status: "planned"}},
		Blueprints: []BlueprintView{
			{ID: 11, Code: "aws-vpc-base", Name: "AWS 基础网络", Category: "network"},
			{ID: 12, Code: "aws-bastion", Name: "AWS Ops Bastion", Category: "compute"},
			{ID: 13, Code: "aws-eks-quickstart", Name: "AWS Cluster Blueprint", Category: "cluster", SchemaJSON: `{"properties":{"network_ref":{"type":"string"}}}`},
		},
		Jobs: []DeploymentJobView{{
			ID:            101,
			Name:          "prod-base-apply",
			BlueprintID:   11,
			BlueprintName: "AWS 基础网络",
			Status:        "succeeded",
			Action:        "apply",
			CreatedAt:     time.Now(),
		}},
	})

	assertWorkbenchStage(t, view, "delivery_objects", "scroll-networks")
	if !view.FoundationComplete {
		t.Fatalf("expected foundation to be complete")
	}
	if !view.ClusterContractReady {
		t.Fatalf("expected cluster contract to be detected")
	}
}

func assertWorkbenchIdentity(t *testing.T, view CloudWorkbenchView) {
	t.Helper()
	if view.ModuleLabel != "基础建设" {
		t.Fatalf("unexpected module label: %q", view.ModuleLabel)
	}
	if view.WorkflowLabel != "基础交付" {
		t.Fatalf("unexpected workflow label: %q", view.WorkflowLabel)
	}
}

func assertWorkbenchStage(t *testing.T, view CloudWorkbenchView, key, primaryAction string) {
	t.Helper()
	if view.Stage.Key != key {
		t.Fatalf("unexpected stage key: got %q want %q", view.Stage.Key, key)
	}
	if view.Stage.PrimaryAction != primaryAction {
		t.Fatalf("unexpected primary action: got %q want %q", view.Stage.PrimaryAction, primaryAction)
	}
}

func assertWorkbenchBlocker(t *testing.T, view CloudWorkbenchView, titlePart string) {
	t.Helper()
	if len(view.Blockers) == 0 {
		t.Fatalf("expected at least one blocker")
	}
	if !strings.Contains(view.Blockers[0].Title, titlePart) {
		t.Fatalf("unexpected first blocker title: %q, want it to contain %q", view.Blockers[0].Title, titlePart)
	}
}
