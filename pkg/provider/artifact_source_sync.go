package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func artifactSourceConfigured(data *ArtifactResourceModel) bool {
	return data.Source != nil && IsKnown(data.Source.Dir)
}

func artifactSourceNeedsUpload(plan, state *ArtifactResourceModel, priorArtifactID, newArtifactID string) bool {
	if !artifactSourceConfigured(plan) {
		return false
	}
	if priorArtifactID != newArtifactID {
		return true
	}
	if state == nil || state.Source == nil || !IsKnown(state.Source.DirHash) {
		return true
	}
	return !plan.Source.DirHash.Equal(state.Source.DirHash)
}

func catalogIDFromModel(data *ArtifactResourceModel) string {
	if data == nil || data.Spec == nil {
		return ""
	}
	for _, group := range data.Spec.ContainerGroups {
		for _, container := range group.Containers {
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				continue
			}
			if IsKnown(ref.CatalogID) {
				return ref.CatalogID.ValueString()
			}
		}
	}
	return ""
}

func catalogVersionIDFromModel(data *ArtifactResourceModel) string {
	if data == nil || data.Spec == nil {
		return ""
	}
	for _, group := range data.Spec.ContainerGroups {
		for _, container := range group.Containers {
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				continue
			}
			if IsKnown(ref.CatalogVersionID) {
				return ref.CatalogVersionID.ValueString()
			}
		}
	}
	return ""
}

// pushArtifactSource uploads absDir. seeded is the matcher to upload with when
// the caller could not write the starter ignore file, nil when the directory is
// the source of truth as usual.
func (r *ArtifactResource) pushArtifactSource(
	ctx context.Context,
	prior *ArtifactResourceModel,
	existingCatalogID string,
	absDir string,
	seeded *ignore.Matcher,
) (*artifactsource.Result, error) {
	opts := artifactsource.Options{
		Dir:       absDir,
		CatalogID: existingCatalogID,
	}
	if prior != nil {
		opts.CatalogVersionID = catalogVersionIDFromModel(prior)
	}

	matcher := seeded
	if matcher == nil {
		var err error
		matcher, err = ignore.New(absDir)
		if err != nil {
			return nil, fmt.Errorf("load ignore rules: %w", err)
		}
	}
	opts.Ignore = matcher.Match

	traceAPICall("PushDirectory")
	return artifactsource.PushDirectory(ctx, r.provider.service.FilesAPI(), opts)
}

// artifactSourceAbsDir resolves source.dir. The ignore file is looked up at that
// path and the fingerprint walks from it, so everything that touches the source
// tree resolves it the same way.
func artifactSourceAbsDir(data *ArtifactResourceModel) (string, error) {
	dir := data.Source.Dir.ValueString()
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve source directory %q: %w", dir, err)
	}

	return absDir, nil
}

// seedArtifactSourceIgnoreFile writes the starter .drignore at absDir when
// generate_ignore is on and the project has neither ignore file. It returns the
// matcher the upload must use in place of reading the directory, nil when the
// directory can be read as usual.
//
// It runs ahead of artifactSourceNeedsUpload rather than inside the upload,
// because plan has already accounted for the file: with generate_ignore on and
// no ignore file present, computeArtifactSourceDirHash folds a synthetic
// .drignore into dir_hash. A tree whose only pending change is that file
// therefore plans as unchanged, the upload is skipped, and the file plan
// promised is never written -- so a config that flips generate_ignore to true,
// or state carried across a provider upgrade, could record generate_ignore =
// true with nothing on disk. Seeding before the gate makes the promise hold
// whether or not anything was uploaded.
//
// A failed write is a warning rather than an error. generate_ignore defaults to
// true, so failing here would break applies that worked before the attribute
// existed: a source.dir mounted read-only in CI, a 0555 tree, or a directory
// sitting at the .drignore name. The upload falls back to the template's own
// patterns, which is the set plan hashed, so the failure costs the user the
// file on disk and nothing else. In particular it does not widen the upload.
func seedArtifactSourceIgnoreFile(absDir string, generateIgnore bool, diags *diag.Diagnostics) *ignore.Matcher {
	if !generateIgnore {
		return nil
	}

	_, err := ignore.WriteDefaultDrignoreIfMissing(absDir)
	if err == nil {
		return nil
	}

	// WriteDefaultDrignoreIfMissing already names the path it failed to write,
	// so this adds the consequence and the ways out, not the path again.
	diags.AddAttributeWarning(
		path.Root("source").AtName("dir"),
		fmt.Sprintf("Could not write %s", ignore.FileName),
		fmt.Sprintf(
			"%s\n\n"+
				"The upload continues with the default patterns, the same set this plan "+
				"hashed, so nothing extra is uploaded. What is missing is the file on disk, "+
				"which means this repeats on every apply. Add a %s to the directory yourself, "+
				"make the directory writable, or set generate_ignore = false.",
			err, ignore.FileName),
	)

	return ignore.FromDefaultTemplate()
}

func (r *ArtifactResource) syncArtifactSource(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
	diags *diag.Diagnostics,
) (*client.Artifact, bool, error) {
	// Seeding reads source.dir before artifactSourceNeedsUpload gets to answer
	// for an unconfigured source, so the check it used to rely on happens here.
	if !artifactSourceConfigured(plan) {
		return artifact, false, nil
	}

	absDir, err := artifactSourceAbsDir(plan)
	if err != nil {
		return nil, false, err
	}

	// Deliberately ahead of the upload gate below: see seedArtifactSourceIgnoreFile.
	seeded := seedArtifactSourceIgnoreFile(absDir, artifactSourceGenerateIgnore(plan), diags)

	if !artifactSourceNeedsUpload(plan, state, priorArtifactID, artifact.ID) {
		return artifact, false, nil
	}

	catalogID := catalogIDFromModel(state)
	if catalogID == "" {
		if ref := client.ExtractCodeRef(artifact); ref != nil {
			catalogID = ref.CatalogID
		}
	}

	pushResult, err := r.pushArtifactSource(ctx, state, catalogID, absDir, seeded)
	if err != nil {
		return nil, false, fmt.Errorf("upload artifact source: %w", err)
	}

	traceAPICall("PatchArtifactCodeRef")
	artifact, err = r.provider.service.PatchArtifactCodeRef(
		ctx,
		artifact.ID,
		pushResult.CatalogID,
		pushResult.CatalogVersionID,
	)
	if err != nil {
		return nil, true, fmt.Errorf("patch artifact code reference: %w", err)
	}

	return artifact, true, nil
}

// syncArtifactSourceAndBuild uploads source when needed, then triggers an image build
// when upload produced new code on a draft artifact with image_build_config.
func (r *ArtifactResource) syncArtifactSourceAndBuild(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
	diags *diag.Diagnostics,
) (*client.Artifact, error) {
	artifact, uploaded, err := r.syncArtifactSource(ctx, plan, state, artifact, priorArtifactID, diags)
	if err != nil {
		return nil, err
	}

	if !artifactBuildNeededAfterUpload(plan, artifact, uploaded) {
		return artifact, nil
	}

	waitForBuild := artifactSourceWaitForBuild(plan)
	repoID := ""
	if artifact.ArtifactRepositoryID != nil {
		repoID = *artifact.ArtifactRepositoryID
	}
	artifact, _, err = r.syncArtifactBuild(ctx, artifact.ID, repoID, waitForBuild, nil)
	if err != nil {
		return artifact, &artifactBuildSyncError{cause: err}
	}

	return artifact, nil
}

func (r *ArtifactResource) rollbackArtifactCreate(ctx context.Context, artifact *client.Artifact, deleteRepository bool) {
	if artifact == nil || !deleteRepository || artifact.ArtifactRepositoryID == nil {
		return
	}
	traceAPICall("DeleteArtifactRepository")
	_ = r.provider.service.DeleteArtifactRepository(ctx, *artifact.ArtifactRepositoryID)
}

// artifactSourcePendingUpload reports whether the planned source tree differs from state.
func artifactSourcePendingUpload(plan, state *ArtifactResourceModel, priorArtifactID string) bool {
	return artifactSourceConfigured(plan) &&
		artifactSourceNeedsUpload(plan, state, priorArtifactID, priorArtifactID)
}

// artifactLockedSourceCloneNeeded is true when a locked artifact needs a draft clone
// before upload (source dir change or spec change that creates a new version).
// Locked artifacts are immutable; the provider clones to draft, uploads, patches code_ref,
// then locks the new version (mirrors CLI guidance in cli/internal/workload/sync/phase1_gather.go).
func artifactLockedSourceCloneNeeded(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	if !artifactSourceConfigured(&plan) {
		return false
	}
	priorArtifactID := state.ArtifactID.ValueString()
	if artifactSourceNeedsUpload(&plan, &state, priorArtifactID, priorArtifactID) {
		return true
	}
	return artifactNeedsNewVersion(plan, state)
}

// artifactSourceDeferLock is true when a draft→locked transition must wait until after source upload.
func artifactSourceDeferLock(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusDraft) {
		return false
	}
	if plan.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	return artifactSourcePendingUpload(&plan, &state, state.ArtifactID.ValueString())
}

// artifactModifyPlanNeedsUnknownArtifactID is true when apply will produce a new artifact version.
func artifactModifyPlanNeedsUnknownArtifactID(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	if plan.Status.ValueString() == string(client.ArtifactStatusDraft) {
		return true
	}
	if artifactNeedsNewVersion(plan, state) {
		return true
	}
	return artifactLockedSourceCloneNeeded(plan, state)
}

// lockArtifact promotes a draft artifact to locked via PATCH {"status": "locked"}.
// Ported from CLI LockArtifact in cli/internal/workload/artifact.go.
func (r *ArtifactResource) lockArtifact(ctx context.Context, artifactID string) (*client.Artifact, error) {
	locked := client.ArtifactStatusLocked
	traceAPICall("PatchArtifact")
	return r.provider.service.PatchArtifact(ctx, artifactID, &client.PatchArtifactRequest{
		Status: &locked,
	})
}

func refreshArtifactSourceDirHash(data *ArtifactResourceModel) {
	if !artifactSourceConfigured(data) {
		return
	}
	dirHash, err := computeArtifactSourceDirHash(data)
	if err == nil {
		data.Source.DirHash = dirHash
	}
}

func artifactSourceGenerateIgnore(data *ArtifactResourceModel) bool {
	if data == nil || data.Source == nil {
		return true
	}
	if data.Source.GenerateIgnore.IsNull() || data.Source.GenerateIgnore.IsUnknown() {
		return true
	}
	return data.Source.GenerateIgnore.ValueBool()
}

// artifactSourceIgnoreDiagnostics reports the two ignore-file conditions the
// matcher can detect but nothing else in an apply would mention: the deprecated
// .wapiignore name still being in effect, and a second ignore file whose
// patterns are silently inert. The first sign of the latter otherwise is a
// .venv on the remote.
//
// These are raised during planning so the user reads them before anything is
// uploaded, rather than after the catalog already has the file.
//
// A matcher that fails to load is not reported here, and which side covers that
// depends on the branch computeArtifactSourceDirHash takes. Everywhere but the
// default path -- generate_ignore on with no ignore file present -- compute
// calls New on this same directory and turns the failure into an attribute
// error, so repeating it would print the problem twice. On the default path
// compute hashes from the template and opens nothing, so plan stays silent by
// design. A directory sitting at the .drignore name, which Locate calls absent,
// lands there; apply names it in the warning seedArtifactSourceIgnoreFile
// raises when the write fails.
func artifactSourceIgnoreDiagnostics(data *ArtifactResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data == nil || data.Source == nil || !IsKnown(data.Source.Dir) {
		return diags
	}

	absDir, err := artifactSourceAbsDir(data)
	if err != nil {
		return diags
	}

	matcher, err := ignore.New(absDir)
	if err != nil {
		return diags
	}

	dirPath := path.Root("source").AtName("dir")
	if notice := matcher.Notice(); notice != "" {
		diags.AddAttributeWarning(dirPath, "Deprecated ignore file name", notice)
	}
	if shadow := matcher.ShadowWarning(); shadow != "" {
		diags.AddAttributeWarning(dirPath, "Ignore file present but not applied", shadow)
	}

	return diags
}

func computeArtifactSourceDirHash(data *ArtifactResourceModel) (types.String, error) {
	hash := types.StringNull()
	if data == nil || data.Source == nil || !IsKnown(data.Source.Dir) {
		return hash, nil
	}

	absDir, err := artifactSourceAbsDir(data)
	if err != nil {
		return hash, err
	}

	generateIgnore := artifactSourceGenerateIgnore(data)
	var matcher *ignore.Matcher
	var extra []artifactsource.LocalFile

	if generateIgnore && ignore.Locate(absDir) == "" {
		matcher = ignore.FromDefaultTemplate()
		sum := sha256.Sum256(ignore.DefaultTemplate)
		extra = []artifactsource.LocalFile{{
			RelPath: ignore.FileName,
			Hash:    hex.EncodeToString(sum[:]),
			Size:    int64(len(ignore.DefaultTemplate)),
		}}
	} else {
		matcher, err = ignore.New(absDir)
		if err != nil {
			return hash, err
		}
	}

	digest, err := artifactsource.FingerprintDirectory(absDir, matcher.Match, extra)
	if err != nil {
		return hash, err
	}

	return types.StringValue(digest), nil
}

func cloneCodeRefModel(ref *ArtifactCodeRefModel) *ArtifactCodeRefModel {
	if ref == nil {
		return nil
	}
	return &ArtifactCodeRefModel{
		CatalogID:        ref.CatalogID,
		CatalogVersionID: ref.CatalogVersionID,
	}
}

func primaryCodeRefFromState(state *ArtifactResourceModel) *ArtifactCodeRefModel {
	if state == nil || state.Spec == nil {
		return nil
	}
	for _, group := range state.Spec.ContainerGroups {
		for _, container := range group.Containers {
			if !artifactContainerIsPrimary(container, group) {
				continue
			}
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				return nil
			}
			if IsKnown(ref.CatalogID) {
				return cloneCodeRefModel(ref)
			}
			return nil
		}
	}
	return nil
}

func applySourceManagedCodeRefsToPlan(plan, state *ArtifactResourceModel, isCreate bool) {
	if !artifactSourceConfigured(plan) || plan.Spec == nil || artifactHasManualCodeRef(plan.Spec) {
		return
	}

	needsUnknown := sourceManagedCodeRefNeedsUnknown(plan, state, isCreate)
	stateCodeRef := primaryCodeRefFromState(state)

	for gi := range plan.Spec.ContainerGroups {
		group := plan.Spec.ContainerGroups[gi]
		for ci := range group.Containers {
			container := &group.Containers[ci]
			if container.ImageBuildConfig == nil {
				continue
			}
			if !artifactContainerIsPrimary(*container, group) {
				continue
			}
			if codeRefManuallySet(imageBuildConfigCodeRef(container.ImageBuildConfig)) {
				continue
			}

			if needsUnknown {
				container.ImageBuildConfig.CodeRef = types.ObjectUnknown(artifactCodeRefAttrTypes())
				continue
			}

			if stateCodeRef != nil {
				_ = setImageBuildConfigCodeRef(container.ImageBuildConfig, cloneCodeRefModel(stateCodeRef))
			}
		}
	}
}

func sourceManagedCodeRefNeedsUnknown(plan, state *ArtifactResourceModel, isCreate bool) bool {
	if isCreate || state == nil {
		return true
	}

	priorArtifactID := state.ArtifactID.ValueString()
	newArtifactID := priorArtifactID
	if !plan.ArtifactID.IsNull() && !plan.ArtifactID.IsUnknown() {
		newArtifactID = plan.ArtifactID.ValueString()
	} else if plan.ArtifactID.IsUnknown() {
		return true
	}

	if artifactSourceNeedsUpload(plan, state, priorArtifactID, newArtifactID) {
		return true
	}

	if state.Status.ValueString() == string(client.ArtifactStatusLocked) {
		// locked→draft or a spec change always yields a new version, so code_ref is unknown.
		// Source-dir changes already returned above via artifactSourceNeedsUpload.
		return plan.Status.ValueString() == string(client.ArtifactStatusDraft) || artifactNeedsNewVersion(*plan, *state)
	}

	return false
}

func codeRefManuallySet(ref *ArtifactCodeRefModel) bool {
	if ref == nil {
		return false
	}
	return IsKnown(ref.CatalogID) || IsKnown(ref.CatalogVersionID)
}

func artifactContainerIsPrimary(container ArtifactContainerModel, group ArtifactContainerGroupModel) bool {
	isPrimary := !container.Primary.IsNull() && !container.Primary.IsUnknown() && container.Primary.ValueBool()
	if !isPrimary && len(group.Containers) == 1 &&
		(container.Primary.IsNull() || container.Primary.IsUnknown()) {
		isPrimary = true
	}
	return isPrimary
}
