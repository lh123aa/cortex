package license

import (
	"testing"
	"time"
)

func TestPlanFeatures(t *testing.T) {
	cases := map[Plan][]string{
		PlanFree:       nil,
		PlanPro:        {"console", "rerank", "auto-index"},
		PlanTeam:       {"console", "audit-log", "rbac-team", "sso-oidc", "backup-s3", "rerank"},
		PlanEnterprise: {"*"},
		PlanTrial:      {"console", "rerank", "auto-index"},
	}
	for plan, want := range cases {
		got := PlanFeatures(plan)
		if len(got) != len(want) {
			t.Errorf("PlanFeatures(%s) = %v, want %v", plan, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("PlanFeatures(%s)[%d] = %q, want %q", plan, i, got[i], want[i])
			}
		}
	}
}

func TestHasFeature(t *testing.T) {
	free := &Info{Plan: PlanFree}
	pro := &Info{Plan: PlanPro}
	team := &Info{Plan: PlanTeam}
	ent := &Info{Plan: PlanEnterprise}

	if free.HasFeature("console") {
		t.Error("free must not have console")
	}
	if !pro.HasFeature("console") {
		t.Error("pro should have console")
	}
	if pro.HasFeature("audit-log") {
		t.Error("pro must not have audit-log")
	}
	if !team.HasFeature("audit-log") {
		t.Error("team should have audit-log")
	}
	if !ent.HasFeature("anything-at-all") {
		t.Error("enterprise wildcard should grant any feature")
	}
	if (*Info)(nil).HasFeature("console") {
		t.Error("nil license must grant nothing")
	}
	if pro.HasFeature("") {
		t.Error("empty feature name must be false")
	}
}

func TestHasFeatureFromPayload(t *testing.T) {
	// 套餐之外的显式 feature（例如一次性试用/定制授权）
	info := &Info{Plan: PlanPro, Features: []string{"sso-oidc"}}
	if !info.HasFeature("sso-oidc") {
		t.Error("explicit payload feature should be granted")
	}
	if info.HasFeature("audit-log") {
		t.Error("unlisted feature must not be granted")
	}
}

func TestDaysLeft(t *testing.T) {
	if got := (&Info{}).DaysLeft(); got != -1 {
		t.Errorf("perpetual DaysLeft = %d, want -1", got)
	}
	expired := &Info{ExpiresAt: time.Now().Add(-time.Hour)}
	if got := expired.DaysLeft(); got != 0 {
		t.Errorf("expired DaysLeft = %d, want 0", got)
	}
	future := &Info{ExpiresAt: time.Now().Add(72 * time.Hour)}
	if got := future.DaysLeft(); got != 2 && got != 3 {
		t.Errorf("future DaysLeft = %d, want 2..3", got)
	}
	if (*Info)(nil).DaysLeft() != -1 {
		t.Error("nil license should report perpetual")
	}
}

func TestTrialIsValidPlan(t *testing.T) {
	if !PlanTrial.Valid() {
		t.Fatal("trial must be a valid plan")
	}
	info := &Info{Plan: PlanTrial, ExpiresAt: time.Now().Add(24 * time.Hour)}
	if !info.IsTrial() {
		t.Fatal("IsTrial should be true")
	}
	if !info.HasFeature("console") {
		t.Fatal("trial should grant pro-level features")
	}
}

func TestTrialLimits(t *testing.T) {
	// 试用期按 Pro 配额，避免试用期反而卡住用户
	got := Limits(PlanTrial)
	pro := Limits(PlanPro)
	if got != pro {
		t.Fatalf("trial limits = %+v, want pro limits %+v", got, pro)
	}
}
