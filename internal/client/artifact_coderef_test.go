package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractCodeRef_PrefersPrimaryWhenSidecarFirst(t *testing.T) {
	primary := true
	notPrimary := false

	artifact := &Artifact{
		Spec: ArtifactSpec{
			ContainerGroups: []ArtifactContainerGroup{
				{
					Containers: []ArtifactContainer{
						{
							Primary: &notPrimary,
							ImageBuildConfig: &ArtifactImageBuildConfig{
								CodeRef: &ArtifactCodeRef{
									DataRobot: ArtifactDataRobotCodeRef{
										CatalogID:        "sidecar-cat",
										CatalogVersionID: "sidecar-ver",
									},
								},
							},
						},
						{
							Primary: &primary,
							ImageBuildConfig: &ArtifactImageBuildConfig{
								CodeRef: &ArtifactCodeRef{
									DataRobot: ArtifactDataRobotCodeRef{
										CatalogID:        "primary-cat",
										CatalogVersionID: "primary-ver",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	codeRef := ExtractCodeRef(artifact)
	if codeRef == nil {
		t.Fatal("expected code ref")
	}
	if codeRef.CatalogID != "primary-cat" {
		t.Fatalf("catalogId = %q, want primary-cat", codeRef.CatalogID)
	}
	if codeRef.CatalogVersionID != "primary-ver" {
		t.Fatalf("catalogVersionId = %q, want primary-ver", codeRef.CatalogVersionID)
	}
}

func TestSetPrimaryCodeRefInRawArtifact(t *testing.T) {
	t.Run("OverwritesPrimaryCodeRef_LeavesSidecarsAlone", func(t *testing.T) {
		raw := map[string]any{
			"spec": map[string]any{
				"containerGroups": []any{
					map[string]any{
						"containers": []any{
							map[string]any{
								"primary": false,
								"imageBuildConfig": map[string]any{
									"codeRef": map[string]any{
										"datarobot": map[string]any{
											"catalogId":        "sidecar-cat",
											"catalogVersionId": "sidecar-ver",
										},
									},
									"dockerfile": map[string]any{"source": "provided"},
								},
								"imageUri": "sidecar-image",
							},
							map[string]any{
								"primary": true,
								"imageBuildConfig": map[string]any{
									"codeRef": map[string]any{
										"datarobot": map[string]any{
											"catalogId":        "old-cat",
											"catalogVersionId": "old-ver",
										},
									},
									"dockerfile": map[string]any{
										"source":                        "generated",
										"executionEnvironmentId":        "env-1",
										"executionEnvironmentVersionId": "env-v-1",
										"entrypoint":                    []any{"python", "agent.py"},
									},
								},
								"imageUri": "primary-image",
							},
						},
					},
				},
			},
		}

		if err := setPrimaryCodeRefInRawArtifact(raw, "new-cat", "new-ver"); err != nil {
			t.Fatalf("setPrimaryCodeRefInRawArtifact: %v", err)
		}

		spec := mustMap(t, raw["spec"])
		groups := mustSlice(t, spec["containerGroups"])
		group := mustMap(t, groups[0])
		containers := mustSlice(t, group["containers"])
		sidecar := mustMap(t, containers[0])
		primary := mustMap(t, containers[1])

		primaryIBC := mustMap(t, primary["imageBuildConfig"])
		codeRef := mustMap(t, primaryIBC["codeRef"])
		dr := mustMap(t, codeRef["datarobot"])
		if dr["catalogId"] != "new-cat" || dr["catalogVersionId"] != "new-ver" {
			t.Fatalf("primary codeRef = %#v, want new-cat/new-ver", dr)
		}

		df := mustMap(t, primaryIBC["dockerfile"])
		if df["source"] != "generated" || df["executionEnvironmentId"] != "env-1" {
			t.Fatalf("primary dockerfile mutated: %#v", df)
		}

		sidecarIBC := mustMap(t, sidecar["imageBuildConfig"])
		sidecarCodeRef := mustMap(t, sidecarIBC["codeRef"])
		sidecarDR := mustMap(t, sidecarCodeRef["datarobot"])
		if sidecarDR["catalogId"] != "sidecar-cat" || sidecarDR["catalogVersionId"] != "sidecar-ver" {
			t.Fatalf("sidecar codeRef changed: %#v", sidecarDR)
		}
	})

	t.Run("FallsBackToFirstContainerWhenNoPrimary", func(t *testing.T) {
		raw := map[string]any{
			"spec": map[string]any{
				"containerGroups": []any{
					map[string]any{
						"containers": []any{
							map[string]any{"imageUri": "first-image"},
							map[string]any{"imageUri": "second-image"},
						},
					},
				},
			},
		}

		if err := setPrimaryCodeRefInRawArtifact(raw, "new-cat", "new-ver"); err != nil {
			t.Fatalf("setPrimaryCodeRefInRawArtifact: %v", err)
		}

		spec := mustMap(t, raw["spec"])
		groups := mustSlice(t, spec["containerGroups"])
		group := mustMap(t, groups[0])
		containers := mustSlice(t, group["containers"])
		first := mustMap(t, containers[0])
		second := mustMap(t, containers[1])

		ibc := mustMap(t, first["imageBuildConfig"])
		codeRef := mustMap(t, ibc["codeRef"])
		dr := mustMap(t, codeRef["datarobot"])
		if dr["catalogId"] != "new-cat" || dr["catalogVersionId"] != "new-ver" {
			t.Fatalf("first container codeRef = %#v", dr)
		}
		if _, ok := second["imageBuildConfig"]; ok {
			t.Fatal("second container must be untouched")
		}
	})

	cases := []struct {
		name    string
		raw     map[string]any
		wantSub string
	}{
		{"missing spec", map[string]any{}, "spec missing"},
		{"spec wrong type", map[string]any{"spec": "not-a-map"}, "spec missing"},
		{"empty containerGroups", map[string]any{"spec": map[string]any{"containerGroups": []any{}}}, "containerGroups missing or empty"},
		{"missing containerGroups", map[string]any{"spec": map[string]any{}}, "containerGroups missing or empty"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := setPrimaryCodeRefInRawArtifact(tc.raw, "cat", "ver")
			if err == nil {
				t.Fatal("expected error")
			}
			if got := err.Error(); !strings.Contains(got, tc.wantSub) {
				t.Fatalf("error = %q, want substring %q", got, tc.wantSub)
			}
		})
	}
}

func TestPatchArtifactCodeRef_GetPatchGet(t *testing.T) {
	getCount := 0
	var patchBody map[string]any

	artifactAfterPatch := map[string]any{
		"id":     "art-1",
		"name":   "app",
		"status": "draft",
		"spec": map[string]any{
			"containerGroups": []any{
				map[string]any{
					"containers": []any{
						map[string]any{
							"primary": true,
							"imageBuildConfig": map[string]any{
								"codeRef": map[string]any{
									"datarobot": map[string]any{
										"catalogId":        "cat-new",
										"catalogVersionId": "ver-new",
									},
								},
								"dockerfile": map[string]any{"source": "provided"},
							},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/artifacts/art-1/":
			getCount++
			w.Header().Set("Content-Type", "application/json")
			if getCount == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":     "art-1",
					"name":   "app",
					"status": "draft",
					"spec": map[string]any{
						"containerGroups": []any{
							map[string]any{
								"containers": []any{
									map[string]any{"primary": true},
								},
							},
						},
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(artifactAfterPatch)
		case r.Method == http.MethodPatch && r.URL.Path == "/artifacts/art-1/":
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read patch body: %v", err)
			}
			if err := json.Unmarshal(bodyBytes, &patchBody); err != nil {
				t.Fatalf("unmarshal patch body: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	artifact, err := svc.PatchArtifactCodeRef(context.Background(), "art-1", "cat-new", "ver-new")
	if err != nil {
		t.Fatalf("PatchArtifactCodeRef: %v", err)
	}

	if getCount != 2 {
		t.Fatalf("GET count = %d, want 2", getCount)
	}

	spec, ok := patchBody["spec"].(map[string]any)
	if !ok {
		t.Fatalf("patch body spec = %#v", patchBody)
	}
	groups := mustSlice(t, spec["containerGroups"])
	group := mustMap(t, groups[0])
	containers := mustSlice(t, group["containers"])
	container := mustMap(t, containers[0])
	ibc := mustMap(t, container["imageBuildConfig"])
	patchCodeRef := mustMap(t, ibc["codeRef"])
	dr := mustMap(t, patchCodeRef["datarobot"])
	if dr["catalogId"] != "cat-new" || dr["catalogVersionId"] != "ver-new" {
		t.Fatalf("patched codeRef = %#v", dr)
	}

	codeRef := ExtractCodeRef(artifact)
	if codeRef == nil || codeRef.CatalogID != "cat-new" || codeRef.CatalogVersionID != "ver-new" {
		t.Fatalf("returned artifact codeRef = %#v", codeRef)
	}
}

func mustMap(t *testing.T, raw any) map[string]any {
	t.Helper()
	m, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T (%#v)", raw, raw)
	}
	return m
}

func mustSlice(t *testing.T, raw any) []any {
	t.Helper()
	s, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T (%#v)", raw, raw)
	}
	return s
}
