package artifactsource

import "sort"

// PushPlan is the incremental upload blueprint: only paths that changed since
// the last successful push (BaseFiles from Terraform state).
type PushPlan struct {
	Uploads []string // added or modified paths
	Deletes []string // removed since last push
	// no downloads here because we're intending a push-only behavior
}

func (p *PushPlan) IsEmpty() bool {
	return len(p.Uploads) == 0 && len(p.Deletes) == 0
}

func (p *PushPlan) sort() {
	sort.Strings(p.Uploads)
	sort.Strings(p.Deletes)
}

// DiffPushOnly compares the previous manifest (Terraform state) to the current
// local tree. Local is the source of truth; this is push-only (no downloads).
func DiffPushOnly(base, local Manifest) *PushPlan {
	plan := &PushPlan{}

	for path, entry := range local {
		prev, ok := base[path]
		if !ok || prev.Hash != entry.Hash {
			plan.Uploads = append(plan.Uploads, path)
		}
	}

	for path := range base {
		if _, ok := local[path]; !ok {
			plan.Deletes = append(plan.Deletes, path)
		}
	}

	plan.sort()
	return plan
}
