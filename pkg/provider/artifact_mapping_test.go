package provider

import (
	"encoding/json"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnvironmentVarModelFromAPI(t *testing.T) {
	cases := []struct {
		name string
		in   client.ArtifactEnvironmentVariable
		want ArtifactEnvironmentVariableModel
	}{
		{
			name: "string source",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceString, Name: "ENV", Value: "production"},
			want: ArtifactEnvironmentVariableModel{
				Source:         types.StringValue("string"),
				Name:           types.StringValue("ENV"),
				Value:          types.StringValue("production"),
				DrCredentialID: types.StringNull(),
				Key:            types.StringNull(),
			},
		},
		{
			name: "credential source",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceCredential, Name: "SECRET", DrCredentialID: "cred-abc", Key: "token"},
			want: ArtifactEnvironmentVariableModel{
				Source:         types.StringValue("dr-credential"),
				Name:           types.StringValue("SECRET"),
				Value:          types.StringNull(),
				DrCredentialID: types.StringValue("cred-abc"),
				Key:            types.StringValue("token"),
			},
		},
		{
			name: "api-key source with explicit name",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceAPIKey, Name: "MY_DR_TOKEN"},
			want: ArtifactEnvironmentVariableModel{
				Source:         types.StringValue("api-key"),
				Name:           types.StringValue("MY_DR_TOKEN"),
				Value:          types.StringNull(),
				DrCredentialID: types.StringNull(),
				Key:            types.StringNull(),
			},
		},
		{
			// The API stores an omitted api-key name as absent (it resolves to
			// DATAROBOT_API_TOKEN at consumption time), so it must read back
			// as null, not "".
			name: "api-key source without name",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceAPIKey},
			want: ArtifactEnvironmentVariableModel{
				Source:         types.StringValue("api-key"),
				Name:           types.StringNull(),
				Value:          types.StringNull(),
				DrCredentialID: types.StringNull(),
				Key:            types.StringNull(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := environmentVarModelFromAPI(tc.in)
			if got != tc.want {
				t.Errorf("environmentVarModelFromAPI(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestArtifactEnvironmentVariableSerialization(t *testing.T) {
	cases := []struct {
		name string
		in   client.ArtifactEnvironmentVariable
		want string
	}{
		{
			name: "api-key without name omits name",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceAPIKey},
			want: `{"source":"api-key"}`,
		},
		{
			name: "api-key with name",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceAPIKey, Name: "MY_DR_TOKEN"},
			want: `{"source":"api-key","name":"MY_DR_TOKEN"}`,
		},
		{
			name: "string source",
			in:   client.ArtifactEnvironmentVariable{Source: client.EnvironmentVariableSourceString, Name: "ENV", Value: "production"},
			want: `{"source":"string","name":"ENV","value":"production"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("marshal(%+v) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}
