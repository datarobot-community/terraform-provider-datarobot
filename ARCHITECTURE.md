# Terraform DataRobot Provider Architecture

This document describes the proposed refined project structure, domain boundaries, and conventions for maintaining and extending the Terraform DataRobot provider.

---
## High-Level Goals
- Clear domain-driven layout (resources & data sources grouped by functional domain)
- Thin Terraform SDK layer (resource CRUD + schema) delegating logic to internal clients & converters
- Separation of concerns: schema, models (flatten/expand), converters, low-level API clients, shared helpers
- Explicit, auditable resource & data source registration in a single provider entrypoint
- No cyclic or lateral (sibling) imports between domain packages

---
## Directory Tree
```text
terraform-provider-datarobot/
├── ARCHITECTURE.md               # (This file)
├── CHANGELOG.md
├── DEVELOPMENT.md
├── LICENSE
├── LOCAL_TESTING_RUNBOOK.md
├── Makefile
├── README.md
├── go.mod
├── go.sum
├── main.go                       # Entrypoint: serves the provider plugin
├── terraform-registry-manifest.json
├── cmd/
│   └── scaffold/
│       └── templates/
├── docs/                         # Generated + manual Terraform docs
│   ├── index.md
│   ├── data-sources/
│   └── resources/
├── examples/                     # Usage examples (provider, resources, workflows)
│   ├── provider/
│   ├── resources/
│   └── workflows/
├── internal/                     # Internal-only (NOT part of public API)
│   ├── client/                   # Low-level HTTP/API & domain-specific clients
│   │   ├── http.go
│   │   ├── auth.go
│   │   ├── retry.go
│   │   ├── instrumentation.go
│   │   └── domains/
│   │       ├── applications_client.go
│   │       ├── datasets_client.go
│   │       ├── deployments_client.go
│   │       ├── environments_client.go
│   │       ├── jobs_client.go
│   │       ├── models_client.go
│   │       ├── notifications_client.go
│   │       └── genai_client.go
│   ├── common/                   # Pure helpers (NO Terraform SDK imports)
│   │   ├── logging.go
│   │   ├── timeutil.go
│   │   ├── validators.go
│   │   ├── diffutils.go
│   │   ├── hashing.go
│   │   └── errors.go
│   ├── converters/               # API <-> provider models (mapping logic)
│   │   ├── deployments.go
│   │   ├── datasets.go
│   │   └── ...
│   └── testutil/                 # Shared test helpers (acceptance/integration)
│       ├── env.go
│       ├── check.go
│       └── fixtures.go
├── pkg/                          # Public provider packages (Terraform surface)
│   ├── provider/                 # Provider definition & registration
│   │   ├── provider.go           # Configure + resources/data sources map
│   │   ├── config.go             # Provider schema + config decoding
│   │   ├── diagnostics.go
│   │   ├── version.go
│   │   └── meta.go               # Accessors (GetClient, etc.)
│   ├── schema/                   # Reusable schema fragments & TF validators
│   │   ├── common_attributes.go
│   │   ├── validators.go         # Wraps internal/common validators
│   │   └── plan_modifiers.go
│   ├── resources/                # Resource implementations (SDK layer)
│   │   ├── application/
│   │   │   ├── application_resource.go
│   │   │   ├── application_source_resource.go
│   │   │   ├── application_source_from_template_resource.go
│   │   │   └── custom_application_from_environment_resource.go
│   │   ├── auth/
│   │   │   ├── api_token_credential_resource.go
│   │   │   ├── basic_credential_resource.go
│   │   │   ├── aws_credential_resource.go
│   │   │   ├── azure_credential_resource.go
│   │   │   └── google_cloud_credential_resource.go
│   │   ├── data/
│   │   │   ├── dataset_from_file_resource.go
│   │   │   ├── dataset_from_url_resource.go
│   │   │   ├── dataset_from_datasource_resource.go
│   │   │   └── datasource_resource.go
│   │   ├── deployment/
│   │   │   ├── deployment_resource.go
│   │   │   ├── deployment_retraining_policy_resource.go
│   │   │   └── prediction_environment_resource.go
│   │   ├── environment/
│   │   │   ├── execution_environment_resource.go
│   │   │   └── custom_application_resource.go
│   │   ├── genai/
│   │   │   ├── llm_blueprint_resource.go
│   │   │   └── custom_model_llm_validation_resource.go
│   │   ├── job/
│   │   │   ├── batch_prediction_job_definition_resource.go
│   │   │   ├── custom_job_resource.go
│   │   │   ├── custom_metric_job_resource.go
│   │   │   └── notebook_resource.go
│   │   ├── metric/
│   │   │   ├── custom_metric_resource.go
│   │   │   └── custom_metric_from_job_resource.go
│   │   ├── model/
│   │   │   ├── custom_model_resource.go
│   │   │   ├── registered_model_from_leaderboard_resource.go
│   │   │   └── global_model_resource.go
│   │   ├── notification/
│   │   │   ├── notification_channel_resource.go
│   │   │   └── notification_policy_resource.go
│   │   └── playground/
│   │       └── playground_resource.go
│   ├── data_sources/             # Data source implementations
│   │   ├── model/
│   │   │   ├── global_model_data_source.go
│   │   │   └── registered_model_from_leaderboard_data_source.go
│   │   ├── environment/
│   │   │   └── execution_environment_data_source.go
│   │   └── ...
│   └── models/                   # Terraform-facing logical models (expand/flatten)
│       ├── application_models.go
│       ├── deployment_models.go
│       ├── dataset_models.go
│       ├── job_models.go
│       ├── model_models.go
│       ├── metric_models.go
│       ├── notification_models.go
│       └── shared.go
├── test/
│   ├── service_test.go
│   ├── assets/
│   │   ├── datarobot_english_documentation_docsassist.zip
│   │   └── golang_prebuilt_environment.tar.gz
│   └── acceptance/
│       ├── provider_test.go
│       └── resources_*.go
├── tools/
│   ├── tools.go
│   └── scripts/
│       ├── generate-docs.sh
│       ├── verify.sh
│       └── move-resources.sh
└── utils/ (OPTIONAL – prefer folding into internal/common or pkg/schema)
```

---
## Layering & Allowed Dependencies
```
internal/common      <- no Terraform SDK imports
internal/client      <- may use internal/common
internal/converters  <- may use internal/common + internal/client + pkg/models
pkg/models           <- pure structs + flatten/expand logic (no HTTP; can use internal/common)
pkg/schema           <- Terraform SDK schema & validators (may wrap internal/common)
pkg/resources        <- Terraform resource definitions (depend on provider, schema, models, converters, client via provider meta)
pkg/data_sources     <- Same pattern as resources
pkg/provider         <- Wires everything; imports resources + data sources + schema + version
```
Forbidden:
- Lateral imports between sibling domain folders in `pkg/resources/` or `pkg/data_sources/`
- Resources importing `internal/client/domains` directly if access goes through higher-level aggregated client (prefer provider meta)
- Terraform SDK imports inside `internal/common`

---
## Domain Mapping Reference
| Domain Folder | Concept Examples |
| ------------- | ---------------- |
| application   | Application sources, templates, custom apps |
| auth          | Credential & secrets management |
| data          | Datasets, datasources, dataset ingestion |
| deployment    | Deployments, prediction environments, retraining policies |
| environment   | Execution/custom environments |
| genai         | LLM blueprints, LLM validation |
| job           | Custom jobs, batch prediction job definitions, notebooks |
| metric        | Custom metrics & derived metrics |
| model         | Models, registered models, global models |
| notification  | Channels & policies |
| playground    | Experimental / sandbox resources |

---
## Adding a New Resource (Checklist)
1. Create file under `pkg/resources/<domain>/<name>_resource.go`
2. Implement `func <PascalName>Resource() resource.Resource` (constructor)
3. Define schema using helpers from `pkg/schema`
4. Use provider meta to obtain client: `client := p.Client()` (accessor in `meta.go`)
5. Keep expand/flatten logic in `pkg/models` + `internal/converters`
6. Register it in `pkg/provider/provider.go` in the `Resources` map
7. Add docs in `docs/resources/<domain>_<name>.md` (or rely on generator)
8. Add example usage in `examples/resources/<domain>/<name>.tf`
9. Add acceptance test scaffold in `test/acceptance/resources_<domain>_<name>_test.go`

---
## Adding a New Data Source
Mirror the resource flow, but place implementation in `pkg/data_sources/<domain>/` and register in `DataSourcesMap` within `provider.go`.

---
## Testing Strategy
| Layer        | Test Type | Location |
| ------------ | --------- | -------- |
| models       | Unit (expand/flatten roundtrip) | `pkg/models/*_test.go` |
| converters   | Unit (API ↔ model mapping)      | `internal/converters/*_test.go` |
| client       | Unit (mock HTTP)                | `internal/client/*_test.go` |
| resources    | Light unit (schema invariants)  | `pkg/resources/..._test.go` |
| provider     | Acceptance (full lifecycle)     | `test/acceptance/` |

---
## Migration Notes
If refactoring an existing monolith:
1. Move low-level HTTP logic into `internal/client`
2. Extract pure helpers (no SDK) into `internal/common`
3. Split large `models.go` into domain-specific files under `pkg/models`
4. Create domain folders and move resource/data source implementations (filenames unchanged)
5. Introduce `pkg/schema` for common attribute builders/validators
6. Update `provider.go` to register with new import paths
7. Run `go vet`, `staticcheck`, and acceptance tests

---
## Naming Conventions
- Files: `<logical_name>_resource.go`, `<logical_name>_data_source.go`
- Constructors: `<PascalName>Resource()` or `<PascalName>DataSource()`
- Expand funcs: `expand<Model>()`
- Flatten funcs: `flatten<Model>()`
- Converter funcs: `ConvertAPI<Model>ToState`, `ConvertStateToAPI<Model>`

---
## Performance & Maintainability Considerations
- Keep resource CRUD thin; heavy logic lives in converters/models
- Centralize retry/backoff at `internal/client`
- Use small, focused structs in `pkg/models` to minimize state drift complexity
- Prefer composition over inheritance-like patterns; avoid global singletons

---
## Future Enhancements (Backlog Ideas)
- Code generation for repetitive schemas
- Unified pagination helper in `internal/common`
- Telemetry hooks (OpenTelemetry) in `internal/client/instrumentation.go`
- Doc generation script integration in CI
- Schema diff linter to catch breaking changes before release

---
## Quick Glossary
| Term      | Meaning |
| --------- | ------- |
| Expand    | Convert Terraform state -> API request struct |
| Flatten   | Convert API response -> Terraform state model |
| Converter | Glue between API models and provider models |
| Domain    | Functional grouping (deployment, model, etc.) |

---
## Questions / Updates
For structural changes, update this file in the same PR to keep architecture documentation consistent with the codebase.

---
_End of document._
