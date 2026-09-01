# Agent code-to-workload: upload local agent source, build from a GenAI execution environment
# (generated Dockerfile), lock the artifact, and deploy a workload with api-key injection.

resource "datarobot_artifact" "agent" {
  name        = "agent-app-wapi"
  description = "Agent workload artifact built from local source via generated Dockerfile"
  type        = "service"
  status      = "locked"

  source = {
    dir = "${path.module}/app"
    # wait_for_build defaults to true: upload, trigger build, poll until image_uri is ready, then lock
  }

  spec = {
    container_groups = [{
      containers = [{
        name    = "agent"
        primary = true
        port    = 8080

        environment_vars = [
          {
            source = "string"
            name   = "LLM_DEFAULT_MODEL"
            value  = "datarobot/azure/gpt-5-5-2026-04-23"
          },
          {
            source = "api-key"
          },
        ]

        image_build_config = {
          dockerfile = {
            source                           = "generated"
            execution_environment_id         = "" # use the execution environment id you want to use
            execution_environment_version_id = "" # use the execution environment version id you want to use
            entrypoint                       = ["sh", "run_server.sh"]
          }
        }

        readiness_probe = {
          path                  = "/health"
          port                  = 8080
          initial_delay_seconds = 10
          period_seconds        = 10
          timeout_seconds       = 5
          failure_threshold     = 6
          scheme                = "HTTP"
        }
      }]
    }]
  }
}

resource "datarobot_workload" "agent" {
  name        = "agent-app-wapi"
  description = "Workload serving the agent artifact"
  importance  = "low"
  artifact_id = datarobot_artifact.agent.artifact_id

  runtime = {
    container_groups = [{
      replica_count    = 1
      resource_bundles = ["cpu.xlarge"]
    }]
  }

  depends_on = [datarobot_artifact.agent]
}

output "artifact_id" {
  value       = datarobot_artifact.agent.artifact_id
  description = "Locked artifact version ID deployed by the workload"
}

output "workload_endpoint" {
  value       = datarobot_workload.agent.endpoint
  description = "Inference endpoint URL for the deployed agent workload"
}
