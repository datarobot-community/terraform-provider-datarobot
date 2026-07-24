package client

import (
	"context"
	"errors"
	"fmt"
)

// ExtractCodeRef returns the DataRobot catalog reference from the artifact's
// primary container, mirroring the write-side selection in setPrimaryCodeRefInRawArtifact.
// Ported from cli/internal/workload/artifact.go (ExtractCodeRef).
func ExtractCodeRef(artifact *Artifact) *ArtifactDataRobotCodeRef {
	if artifact == nil {
		return nil
	}

	for _, group := range artifact.Spec.ContainerGroups {
		for _, container := range group.Containers {
			if container.Primary == nil || !*container.Primary {
				continue
			}

			return codeRefFromContainer(container)
		}
	}

	if len(artifact.Spec.ContainerGroups) == 0 {
		return nil
	}

	if len(artifact.Spec.ContainerGroups[0].Containers) == 0 {
		return nil
	}

	return codeRefFromContainer(artifact.Spec.ContainerGroups[0].Containers[0])
}

func codeRefFromContainer(container ArtifactContainer) *ArtifactDataRobotCodeRef {
	if container.ImageBuildConfig == nil || container.ImageBuildConfig.CodeRef == nil {
		return nil
	}

	return &container.ImageBuildConfig.CodeRef.DataRobot
}

// PatchArtifactCodeRef updates the primary container's imageBuildConfig.codeRef
// after a Files API upload. GETs the artifact as raw JSON, mutates only the
// selected container's codeRef, and PATCHes {"spec": ...} to preserve fields
// not modeled in Go structs. Ported from cli/internal/workload/artifact.go
// (PatchArtifactCodeRef).
func (s *ServiceImpl) PatchArtifactCodeRef(
	ctx context.Context,
	artifactID, catalogID, catalogVersionID string,
) (*Artifact, error) {
	path := "/artifacts/" + artifactID + "/"

	raw, err := Get[map[string]any](s.client, ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetch artifact for codeRef update: %w", err)
	}

	if err := setPrimaryCodeRefInRawArtifact(*raw, catalogID, catalogVersionID); err != nil {
		return nil, err
	}

	body := map[string]any{"spec": (*raw)["spec"]}
	if _, err := Patch[CreateVoidResponse](s.client, ctx, path, body); err != nil {
		return nil, err
	}

	return Get[Artifact](s.client, ctx, path)
}

func setPrimaryCodeRefInRawArtifact(raw map[string]any, catalogID, catalogVersionID string) error {
	spec, ok := raw["spec"].(map[string]any)
	if !ok {
		return errors.New("artifact: spec missing or wrong type")
	}

	groups, ok := spec["containerGroups"].([]any)
	if !ok || len(groups) == 0 {
		return errors.New("artifact: spec.containerGroups missing or empty")
	}

	codeRef := map[string]any{
		"datarobot": map[string]any{
			"catalogId":        catalogID,
			"catalogVersionId": catalogVersionID,
		},
	}

	if found := assignToPrimaryContainer(groups, codeRef); found {
		return nil
	}

	// Mirror ExtractCodeRef's [0][0] fallback when no container is flagged primary.
	return assignToFirstContainer(groups, codeRef)
}

func assignToPrimaryContainer(groups []any, codeRef map[string]any) bool {
	for _, g := range groups {
		group, ok := g.(map[string]any)
		if !ok {
			continue
		}

		containers, ok := group["containers"].([]any)
		if !ok {
			continue
		}

		for _, c := range containers {
			container, ok := c.(map[string]any)
			if !ok {
				continue
			}

			if isPrimaryContainer(container) {
				setImageBuildConfigCodeRef(container, codeRef)

				return true
			}
		}
	}

	return false
}

func assignToFirstContainer(groups []any, codeRef map[string]any) error {
	firstGroup, ok := groups[0].(map[string]any)
	if !ok {
		return errors.New("artifact: spec.containerGroups[0] missing or wrong type")
	}

	containers, ok := firstGroup["containers"].([]any)
	if !ok || len(containers) == 0 {
		return errors.New("artifact: spec.containerGroups[0].containers missing or empty")
	}

	firstContainer, ok := containers[0].(map[string]any)
	if !ok {
		return errors.New("artifact: spec.containerGroups[0].containers[0] missing or wrong type")
	}

	setImageBuildConfigCodeRef(firstContainer, codeRef)

	return nil
}

// setImageBuildConfigCodeRef preserves any existing dockerfile config and
// seeds a "provided" Dockerfile default when imageBuildConfig is absent
// (server requires a dockerfile on the imageBuildConfig).
// Ported from cli/internal/workload/artifact.go.
func setImageBuildConfigCodeRef(container map[string]any, codeRef map[string]any) {
	ibc, ok := container["imageBuildConfig"].(map[string]any)
	if !ok || ibc == nil {
		ibc = map[string]any{
			"dockerfile": map[string]any{
				"source": "provided",
			},
		}
	}

	ibc["codeRef"] = codeRef
	container["imageBuildConfig"] = ibc
}

func isPrimaryContainer(container map[string]any) bool {
	primary, ok := container["primary"].(bool)

	return ok && primary
}
