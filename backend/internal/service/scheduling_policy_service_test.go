//go:build unit

package service

import (
	"context"
	"testing"
	"time"
)

// ---------- stubs ----------

type stubSchedulingPolicyRepo struct {
	policies   []*SchedulingPolicy
	actions    []*SchedulingActionRecord
	nextActive int64
}

func (r *stubSchedulingPolicyRepo) Create(_ context.Context, p *SchedulingPolicy) error {
	r.policies = append(r.policies, p)
	return nil
}
func (r *stubSchedulingPolicyRepo) GetByID(_ context.Context, id int64) (*SchedulingPolicy, error) {
	for _, p := range r.policies {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, ErrSchedulingPolicyNotFound
}
func (r *stubSchedulingPolicyRepo) Update(_ context.Context, _ *SchedulingPolicy) error { return nil }
func (r *stubSchedulingPolicyRepo) Delete(_ context.Context, _ int64) error            { return nil }
func (r *stubSchedulingPolicyRepo) List(_ context.Context, _ SchedulingPolicyListParams) ([]*SchedulingPolicy, int64, error) {
	return r.policies, int64(len(r.policies)), nil
}
func (r *stubSchedulingPolicyRepo) ListEnabledByMonitorID(_ context.Context, monitorID int64) ([]*SchedulingPolicy, error) {
	var out []*SchedulingPolicy
	for _, p := range r.policies {
		if p.Enabled && p.MonitorID == monitorID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *stubSchedulingPolicyRepo) InsertAction(_ context.Context, a *SchedulingActionRecord) error {
	r.nextActive++
	a.ID = r.nextActive
	a.CreatedAt = time.Now()
	r.actions = append(r.actions, a)
	return nil
}
func (r *stubSchedulingPolicyRepo) ListActions(_ context.Context, _ SchedulingActionListParams) ([]*SchedulingActionRecord, int64, error) {
	return r.actions, int64(len(r.actions)), nil
}
func (r *stubSchedulingPolicyRepo) ListUnrestoredActions(_ context.Context, policyID int64) ([]*SchedulingActionRecord, error) {
	var out []*SchedulingActionRecord
	for _, a := range r.actions {
		if a.PolicyID == policyID && !a.Restored && a.Action != SchedulingActionRecover {
			out = append(out, a)
		}
	}
	return out, nil
}
func (r *stubSchedulingPolicyRepo) MarkActionsRestored(_ context.Context, ids []int64) error {
	idSet := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	for _, a := range r.actions {
		if _, ok := idSet[a.ID]; ok {
			a.Restored = true
		}
	}
	return nil
}
func (r *stubSchedulingPolicyRepo) LatestTriggerAt(_ context.Context, policyID int64) (*time.Time, error) {
	var latest *time.Time
	for _, a := range r.actions {
		if a.PolicyID == policyID && a.Action != SchedulingActionRecover {
			t := a.CreatedAt
			latest = &t
		}
	}
	return latest, nil
}

type stubSchedulingAccountRepo struct {
	accounts       map[int64]*Account
	pausedUntil    map[int64]time.Time
	pauseCleared   map[int64]bool
	priorityWrites map[int64]int
}

func newStubSchedulingAccountRepo(accounts ...*Account) *stubSchedulingAccountRepo {
	m := make(map[int64]*Account)
	for _, a := range accounts {
		m[a.ID] = a
	}
	return &stubSchedulingAccountRepo{
		accounts:       m,
		pausedUntil:    make(map[int64]time.Time),
		pauseCleared:   make(map[int64]bool),
		priorityWrites: make(map[int64]int),
	}
}

func (r *stubSchedulingAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return a, nil
}
func (r *stubSchedulingAccountRepo) SetTempUnschedulable(_ context.Context, id int64, until time.Time, _ string) error {
	r.pausedUntil[id] = until
	return nil
}
func (r *stubSchedulingAccountRepo) ClearTempUnschedulable(_ context.Context, id int64) error {
	r.pauseCleared[id] = true
	return nil
}
func (r *stubSchedulingAccountRepo) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	for _, id := range ids {
		if updates.Priority != nil {
			r.priorityWrites[id] = *updates.Priority
			if a, ok := r.accounts[id]; ok {
				a.Priority = *updates.Priority
			}
		}
	}
	return int64(len(ids)), nil
}

// ---------- helpers ----------

func testPolicy() *SchedulingPolicy {
	return &SchedulingPolicy{
		ID:                          1,
		Name:                        "policy",
		Enabled:                     true,
		MonitorID:                   7,
		AccountIDs:                  []int64{100},
		TriggerConsecutiveFailures:  2,
		ActionType:                  SchedulingActionPause,
		PauseMinutes:                30,
		PriorityDelta:               10,
		RecoverConsecutiveSuccesses: 2,
		CooldownMinutes:             0,
	}
}

func testMonitor() *ChannelMonitor {
	return &ChannelMonitor{ID: 7, Name: "mon", PrimaryModel: "gpt-4o"}
}

func failedResults() []*CheckResult {
	return []*CheckResult{{Model: "gpt-4o", Status: "failed", Message: "boom", CheckedAt: time.Now()}}
}

func okResults() []*CheckResult {
	return []*CheckResult{{Model: "gpt-4o", Status: "operational", CheckedAt: time.Now()}}
}

// ---------- tests ----------

func TestSchedulingPolicyPauseAfterConsecutiveFailures(t *testing.T) {
	repo := &stubSchedulingPolicyRepo{policies: []*SchedulingPolicy{testPolicy()}}
	accounts := newStubSchedulingAccountRepo(&Account{ID: 100, Priority: 50})
	svc := NewSchedulingPolicyService(repo, accounts)
	ctx := context.Background()

	svc.OnCheckResults(ctx, testMonitor(), failedResults())
	if len(accounts.pausedUntil) != 0 {
		t.Fatal("should not pause after single failure")
	}
	svc.OnCheckResults(ctx, testMonitor(), failedResults())
	if _, ok := accounts.pausedUntil[100]; !ok {
		t.Fatal("expected account 100 paused after 2 consecutive failures")
	}
	if len(repo.actions) != 1 || repo.actions[0].Action != SchedulingActionPause {
		t.Fatalf("expected 1 pause action, got %+v", repo.actions)
	}
}

func TestSchedulingPolicyRecoverAfterConsecutiveSuccesses(t *testing.T) {
	repo := &stubSchedulingPolicyRepo{policies: []*SchedulingPolicy{testPolicy()}}
	accounts := newStubSchedulingAccountRepo(&Account{ID: 100, Priority: 50})
	svc := NewSchedulingPolicyService(repo, accounts)
	ctx := context.Background()

	svc.OnCheckResults(ctx, testMonitor(), failedResults())
	svc.OnCheckResults(ctx, testMonitor(), failedResults())
	svc.OnCheckResults(ctx, testMonitor(), okResults())
	if accounts.pauseCleared[100] {
		t.Fatal("should not recover after single success")
	}
	svc.OnCheckResults(ctx, testMonitor(), okResults())
	if !accounts.pauseCleared[100] {
		t.Fatal("expected pause cleared after 2 consecutive successes")
	}
	var recovered bool
	for _, a := range repo.actions {
		if a.Action == SchedulingActionRecover {
			recovered = true
		}
		if a.Action == SchedulingActionPause && !a.Restored {
			t.Fatal("pause action should be marked restored")
		}
	}
	if !recovered {
		t.Fatal("expected recover audit record")
	}
}

func TestSchedulingPolicyDeprioritizeAndRestore(t *testing.T) {
	p := testPolicy()
	p.ActionType = SchedulingActionDeprioritize
	p.PriorityDelta = 25
	repo := &stubSchedulingPolicyRepo{policies: []*SchedulingPolicy{p}}
	accounts := newStubSchedulingAccountRepo(&Account{ID: 100, Priority: 50})
	svc := NewSchedulingPolicyService(repo, accounts)
	ctx := context.Background()

	svc.OnCheckResults(ctx, testMonitor(), failedResults())
	svc.OnCheckResults(ctx, testMonitor(), failedResults())
	if got := accounts.priorityWrites[100]; got != 75 {
		t.Fatalf("expected priority 75, got %d", got)
	}
	svc.OnCheckResults(ctx, testMonitor(), okResults())
	svc.OnCheckResults(ctx, testMonitor(), okResults())
	if got := accounts.priorityWrites[100]; got != 50 {
		t.Fatalf("expected priority restored to 50, got %d", got)
	}
}

func TestSchedulingPolicyLatencyTrigger(t *testing.T) {
	p := testPolicy()
	p.TriggerConsecutiveFailures = 1
	p.TriggerLatencyMs = 1000
	repo := &stubSchedulingPolicyRepo{policies: []*SchedulingPolicy{p}}
	accounts := newStubSchedulingAccountRepo(&Account{ID: 100, Priority: 50})
	svc := NewSchedulingPolicyService(repo, accounts)
	ctx := context.Background()

	slow := 1500
	svc.OnCheckResults(ctx, testMonitor(), []*CheckResult{
		{Model: "gpt-4o", Status: "operational", LatencyMs: &slow, CheckedAt: time.Now()},
	})
	if _, ok := accounts.pausedUntil[100]; !ok {
		t.Fatal("expected latency breach to trigger pause")
	}
}

func TestSchedulingPolicyNoDuplicateTrigger(t *testing.T) {
	repo := &stubSchedulingPolicyRepo{policies: []*SchedulingPolicy{testPolicy()}}
	accounts := newStubSchedulingAccountRepo(&Account{ID: 100, Priority: 50})
	svc := NewSchedulingPolicyService(repo, accounts)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.OnCheckResults(ctx, testMonitor(), failedResults())
	}
	pauses := 0
	for _, a := range repo.actions {
		if a.Action == SchedulingActionPause {
			pauses++
		}
	}
	if pauses != 1 {
		t.Fatalf("expected exactly 1 pause action, got %d", pauses)
	}
}

func TestValidateSchedulingPolicy(t *testing.T) {
	p := testPolicy()
	if err := validateSchedulingPolicy(p); err != nil {
		t.Fatalf("valid policy rejected: %v", err)
	}
	bad := testPolicy()
	bad.ActionType = "nuke"
	if err := validateSchedulingPolicy(bad); err == nil {
		t.Fatal("expected invalid action_type error")
	}
	bad2 := testPolicy()
	bad2.AccountIDs = nil
	if err := validateSchedulingPolicy(bad2); err == nil {
		t.Fatal("expected account_ids required error")
	}
}
