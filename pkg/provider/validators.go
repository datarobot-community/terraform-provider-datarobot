package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func ImportanceValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"LOW",
			"MODERATE",
			"HIGH",
			"CRITICAL",
		),
	}
}

func FairnessMetricSetValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"proportionalParity",
			"equalParity",
			"predictionBalance",
			"trueFavorableAndUnfavorableRateParity",
			"favorableAndUnfavorablePredictiveValueParity",
		),
	}
}

func FeatureSelectionValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"auto",
			"manual",
		),
	}
}

func Float64ZeroToOneValidators() []validator.Float64 {
	return []validator.Float64{
		float64validator.Between(0.0, 1.0),
	}
}

func BatchCountValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.OneOf(
			1,
			5,
			10,
			50,
			100,
			1000,
			10000,
		),
	}
}

func TimelinessFrequencyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"T2H",
			"P1D",
			"P7D",
			"P30D",
			"P90D",
			"P180D",
			"P365D",
			"ALL",
		),
	}
}

func BatchJobsPriorityValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"low",
			"medium",
			"high",
		),
	}
}

func ExecutionEnvironmentUseCasesValidators() []validator.List {
	return []validator.List{
		listvalidator.ValueStringsAre(
			stringvalidator.OneOf(
				"customApplication",
				"customModel",
				"notebook",
			),
		),
	}
}

func ProgrammingLanguageValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"python",
			"java",
			"julia",
			"r",
			"legacy",
			"other",
		),
	}
}

func DataStoreTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"jdbc",
			"dr-connector-v1",
			"dr-database-v1",
		),
	}
}

func DatasetCategoryValidators() []validator.List {
	return []validator.List{
		listvalidator.ValueStringsAre(
			stringvalidator.OneOf(
				"BATCH_PREDICTIONS",
				"MULTI_SERIES_CALENDAR",
				"PREDICTION",
				"SAMPLE",
				"SINGLE_SERIES_CALENDAR",
				"TRAINING",
			),
		),
	}
}

func EgressNetworkPolicyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"public",
			"none",
		),
	}
}

func TimeFormatValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"%m/%d/%Y",
			"%m/%d/%y",
			"%d/%m/%y",
			"%m-%d-%Y",
			"%m-%d-%y",
			"%Y/%m/%d",
			"%Y-%m-%d",
			"%Y-%m-%d %H:%M:%S",
			"%Y/%m/%d %H:%M:%S",
			"%Y.%m.%d %H:%M:%S",
			"%Y-%m-%d %H:%M",
			"%Y/%m/%d %H:%M",
			"%y/%m/%d",
			"%y-%m-%d",
			"%y-%m-%d %H:%M:%S",
			"%y.%m.%d %H:%M:%S",
			"%y/%m/%d %H:%M:%S",
			"%y-%m-%d %H:%M",
			"%y.%m.%d %H:%M",
			"%y/%m/%d %H:%M",
			"%m/%d/%Y %H:%M",
			"%m/%d/%y %H:%M",
			"%d/%m/%Y %H:%M",
			"%d/%m/%y %H:%M",
			"%m-%d-%Y %H:%M",
			"%m-%d-%y %H:%M",
			"%d-%m-%Y %H:%M",
			"%d-%m-%y %H:%M",
			"%m.%d.%Y %H:%M",
			"%m/%d.%y %H:%M",
			"%d.%m.%Y %H:%M",
			"%d.%m.%y %H:%M",
			"%m/%d/%Y %H:%M:%S",
			"%m/%d/%y %H:%M:%S",
			"%m-%d-%Y %H:%M:%S",
			"%m-%d-%y %H:%M:%S",
			"%m.%d.%Y %H:%M:%S",
			"%m.%d.%y %H:%M:%S",
			"%d/%m/%Y %H:%M:%S",
			"%d/%m/%y %H:%M:%S",
			"%Y-%m-%d %H:%M:%S.%f",
			"%y-%m-%d %H:%M:%S.%f",
			"%Y-%m-%dT%H:%M:%S.%fZ",
			"%y-%m-%dT%H:%M:%S.%fZ",
			"%Y-%m-%dT%H:%M:%S.%f",
			"%y-%m-%dT%H:%M:%S.%f",
			"%Y-%m-%dT%H:%M:%S",
			"%y-%m-%dT%H:%M:%S",
			"%Y-%m-%dT%H:%M:%SZ",
			"%y-%m-%dT%H:%M:%SZ",
			"%Y.%m.%d %H:%M:%S.%f",
			"%y.%m.%d %H:%M:%S.%f",
			"%Y.%m.%dT%H:%M:%S.%fZ",
			"%y.%m.%dT%H:%M:%S.%fZ",
			"%Y.%m.%dT%H:%M:%S.%f",
			"%y.%m.%dT%H:%M:%S.%f",
			"%Y.%m.%dT%H:%M:%S",
			"%y.%m.%dT%H:%M:%S",
			"%Y.%m.%dT%H:%M:%SZ",
			"%y.%m.%dT%H:%M:%SZ",
			"%Y%m%d",
			"%m %d %Y %H %M %S",
			"%m %d %y %H %M %S",
			"%H:%M",
			"%M:%S",
			"%H:%M:%S",
			"%Y %m %d %H %M %S",
			"%y %m %d %H %M %S",
			"%Y %m %d",
			"%y %m %d",
			"%d/%m/%Y",
			"%Y-%d-%m",
			"%y-%d-%m",
			"%Y/%d/%m %H:%M:%S.%f",
			"%Y/%d/%m %H:%M:%S.%fZ",
			"%Y/%m/%d %H:%M:%S.%f",
			"%Y/%m/%d %H:%M:%S.%fZ",
			"%y/%d/%m %H:%M:%S.%f",
			"%y/%d/%m %H:%M:%S.%fZ",
			"%y/%m/%d %H:%M:%S.%f",
			"%y/%m/%d %H:%M:%S.%fZ",
			"%m.%d.%Y",
			"%m.%d.%y",
			"%d.%m.%y",
			"%d.%m.%Y",
			"%Y.%m.%d",
			"%Y.%d.%m",
			"%y.%m.%d",
			"%y.%d.%m",
			"%Y-%m-%d %I:%M:%S %p",
			"%Y/%m/%d %I:%M:%S %p",
			"%Y.%m.%d %I:%M:%S %p",
			"%Y-%m-%d %I:%M %p",
			"%Y/%m/%d %I:%M %p",
			"%y-%m-%d %I:%M:%S %p",
			"%y.%m.%d %I:%M:%S %p",
			"%y/%m/%d %I:%M:%S %p",
			"%y-%m-%d %I:%M %p",
			"%y.%m.%d %I:%M %p",
			"%y/%m/%d %I:%M %p",
			"%m/%d/%Y %I:%M %p",
			"%m/%d/%y %I:%M %p",
			"%d/%m/%Y %I:%M %p",
			"%d/%m/%y %I:%M %p",
			"%m-%d-%Y %I:%M %p",
			"%m-%d-%y %I:%M %p",
			"%d-%m-%Y %I:%M %p",
			"%d-%m-%y %I:%M %p",
			"%m.%d.%Y %I:%M %p",
			"%m/%d.%y %I:%M %p",
			"%d.%m.%Y %I:%M %p",
			"%d.%m.%y %I:%M %p",
			"%m/%d/%Y %I:%M:%S %p",
			"%m/%d/%y %I:%M:%S %p",
			"%m-%d-%Y %I:%M:%S %p",
			"%m-%d-%y %I:%M:%S %p",
			"%m.%d.%Y %I:%M:%S %p",
			"%m.%d.%y %I:%M:%S %p",
			"%d/%m/%Y %I:%M:%S %p",
			"%d/%m/%y %I:%M:%S %p",
			"%Y-%m-%d %I:%M:%S.%f %p",
			"%y-%m-%d %I:%M:%S.%f %p",
			"%Y-%m-%dT%I:%M:%S.%fZ %p",
			"%y-%m-%dT%I:%M:%S.%fZ %p",
			"%Y-%m-%dT%I:%M:%S.%f %p",
			"%y-%m-%dT%I:%M:%S.%f %p",
			"%Y-%m-%dT%I:%M:%S %p",
			"%y-%m-%dT%I:%M:%S %p",
			"%Y-%m-%dT%I:%M:%SZ %p",
			"%y-%m-%dT%I:%M:%SZ %p",
			"%Y.%m.%d %I:%M:%S.%f %p",
			"%y.%m.%d %I:%M:%S.%f %p",
			"%Y.%m.%dT%I:%M:%S.%fZ %p",
			"%y.%m.%dT%I:%M:%S.%fZ %p",
			"%Y.%m.%dT%I:%M:%S.%f %p",
			"%y.%m.%dT%I:%M:%S.%f %p",
			"%Y.%m.%dT%I:%M:%S %p",
			"%y.%m.%dT%I:%M:%S %p",
			"%Y.%m.%dT%I:%M:%SZ %p",
			"%y.%m.%dT%I:%M:%SZ %p",
			"%m %d %Y %I %M %S %p",
			"%m %d %y %I %M %S %p",
			"%I:%M %p",
			"%I:%M:%S %p",
			"%Y %m %d %I %M %S %p",
			"%y %m %d %I %M %S %p",
			"%Y/%d/%m %I:%M:%S.%f %p",
			"%Y/%d/%m %I:%M:%S.%fZ %p",
			"%Y/%m/%d %I:%M:%S.%f %p",
			"%Y/%m/%d %I:%M:%S.%fZ %p",
			"%y/%d/%m %I:%M:%S.%f %p",
			"%y/%d/%m %I:%M:%S.%fZ %p",
			"%y/%m/%d %I:%M:%S.%f %p",
			"%y/%m/%d %I:%M:%S.%fZ %p",
		),
	}
}

func DirectionalityValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"lowerIsBetter",
			"higherIsBetter",
		),
	}
}

func ChannelTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"DataRobotCustomJob",
			"DataRobotGroup",
			"DataRobotUser",
			"Database",
			"Email",
			"InApp",
			"InsightsComputations",
			"MSTeams",
			"Slack",
			"Webhook",
		),
	}
}

func ContentTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"application/json",
			"application/x-www-form-urlencoded",
		),
	}
}

func LanguageCodeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"en",
			"es_419",
			"fr",
			"ja",
			"ko",
			"ptBR",
		),
	}
}

func LlmIDValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"azure-openai-gpt-3.5-turbo",
			"azure-openai-gpt-3.5-turbo-16k",
			"azure-openai-gpt-4",
			"azure-openai-gpt-4-32k",
			"azure-openai-gpt-4-turbo",
			"azure-openai-gpt-4-o",
			"azure-openai-gpt-4-o-mini",
			"amazon-titan",
			"amazon-nova-micro",
			"amazon-nova-lite",
			"amazon-nova-pro",
			"anthropic-claude-2",
			"anthropic-claude-3-haiku",
			"anthropic-claude-3-sonnet",
			"anthropic-claude-3-opus",
			"anthropic-claude-3.5-sonnet-v1",
			"amazon-anthropic-claude-3.5-sonnet-v2",
			"google-bison",
			"google-gemini-1.5-flash",
			"google-gemini-1.5-pro",
			"custom-model",
		),
	}
}

func TimeUnitValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"MILLISECOND",
			"SECOND",
			"MINUTE",
			"HOUR",
			"DAY",
			"WEEK",
			"MONTH",
			"QUARTER",
			"YEAR",
			"ROW",
		),
	}
}

func TreatAsExponentialValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"auto",
			"never",
			"always",
		),
	}
}

func RetrainingPolicyActionValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"create_challenger",
			"create_model_package",
			"model_replacement"),
	}
}

func RetrainingPolicyFeatureListStrategyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"same_as_champion",
			"informative_features",
		),
	}
}

func RetrainingPolicyModelSelectionStrategyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"autopilot_recommended",
			"same_blueprint",
			"same_hyperparameters",
			"custom_job",
		),
	}
}

func RetrainingPolicyTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"schedule",
			"data_drift_decline",
			"accuracy_decline",
			"custom_job",
			"None",
		),
	}
}

func RetrainingPolicyModelSelectionMetricValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"Accuracy",
			"AUC",
			"Balanced Accuracy",
			"FVE Binomial",
			"Gini Norm",
			"Kolmogorov-Smirnov",
			"LogLoss",
			"Rate@Top5%",
			"Rate@Top10%",
			"TPR",
			"FPR",
			"TNR",
			"PPV",
			"NPV",
			"F1",
			"MCC",
			"FVE Gamma",
			"FVE Poisson",
			"FVE Tweedie",
			"Gamma Deviance",
			"MAE",
			"MAPE",
			"Poisson Deviance",
			"R Squared",
			"RMSE",
			"RMSLE",
			"Tweedie Deviance",
		),
	}
}

func AutopilotModeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"quick",
			"comprehensive",
			"auto",
		),
	}
}

func CVMethodValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"RandomCV",
			"StratifiedCV",
		),
	}
}

func ModelValidationTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"CV",
			"TVH",
		),
	}
}

func ProjectOptionsStrategyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"same_as_champion",
			"override_champion",
			"custom",
		),
	}
}

func CustomModelTargetTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"Binary",
			"Regression",
			"Multiclass",
			"Anomaly",
			"Transform",
			"TextGeneration",
			"GeoPoint",
			"Unstructured",
			"VectorDatabase",
			"AgenticWorkflow",
		),
	}
}

func CustomModelNetworkEgressPolicyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"PUBLIC",
			"NONE",
		),
	}
}

func PredictionEnvironmentPlatformValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"aws",
			"gcp",
			"azure",
			"onPremise",
			"datarobot",
			"datarobotServerless",
			"openShift",
			"other",
			"snowflake",
			"sapAiCore",
		),
	}
}

func PredictionEnvironmentSupportedModelFormatsValidators() []validator.List {
	return []validator.List{
		listvalidator.ValueStringsAre(
			stringvalidator.OneOf(
				"aws",
				"gcp",
				"azure",
				"onPremise",
				"datarobot",
				"datarobotServerless",
				"openShift",
				"other",
				"snowflake",
				"sapAiCore",
			),
		),
	}
}

func CustomModelLLMTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"openAi",
			"azureOpenAi",
			"google",
			"amazon",
			"datarobot",
		),
		stringvalidator.AlsoRequires(
			path.MatchRelative().AtParent().AtName("openai_credential"),
		),
	}
}

func GuardInterventionActionValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"block",
			"report",
			"replace",
		),
	}
}

func GuardStagesValidators() []validator.List {
	return []validator.List{
		listvalidator.ValueStringsAre(
			stringvalidator.OneOf(
				"prompt",
				"response",
			),
		),
	}
}

func RuntimeParameterTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"boolean",
			"credential",
			"deployment",
			"numeric",
			"string",
		),
	}
}

func EmbeddingModelValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"intfloat/e5-large-v2",
			"intfloat/e5-base-v2",
			"intfloat/multilingual-e5-base",
			"intfloat/multilingual-e5-small",
			"sentence-transformers/all-MiniLM-L6-v2",
			"jinaai/jina-embedding-t-en-v1",
			"jinaai/jina-embedding-s-en-v2",
			"cl-nagoya/sup-simcse-ja-base",
		),
	}
}

func ChunkingMethodValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"recursive",
			"semantic",
		),
	}
}

func NotificationPolicyChannelScopeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"organization",
			"Organization",
			"ORGANIZATION",
			"entity",
			"Entity",
			"ENTITY",
			"template",
			"Template",
			"TEMPLATE",
		),
	}
}

func NotificationRelatedEntityTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"deployment",
			"Deployment",
			"DEPLOYMENT",
		),
	}
}

func NotificationPolicyEventGroupValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"secure_config.all",
			"comment.all",
			"dataset.all",
			"deployment_prediction_explanations_computation.all",
			"model_deployments.critical_health",
			"model_deployments.critical_frequent_health_change",
			"model_deployments.frequent_health_change",
			"model_deployments.health",
			"inference_endpoints.health",
			"model_deployments.management_agent",
			"model_deployments.management_agent_health",
			"prediction_request.all",
			"challenger_management.all",
			"challenger_replay.all",
			"model_deployments.all",
			"project.all",
			"perma_delete_project.all",
			"users_delete.all",
			"applications.all",
			"model_version.stage_transitions",
			"model_version.all",
			"batch_predictions.all",
			"change_requests.all",
			"insights_computation.all",
			"notebook_schedule.all",
			"monitoring.all",
		),
	}
}

func NotificationPolicyEventTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"secure_config.created",
			"secure_config.deleted",
			"secure_config.shared",
			"comment.created",
			"comment.updated",
			"dataset.created",
			"dataset.registered",
			"dataset.deleted",
			"datasets.deleted",
			"datasetrelationship.created",
			"dataset.shared",
			"datasets.shared",
			"misc.asset_access_request",
			"misc.webhook_connection_test",
			"misc.webhook_resend",
			"misc.email_verification",
			"monitoring.spooler_channel_base",
			"monitoring.spooler_channel_red",
			"monitoring.spooler_channel_green",
			"monitoring.external_model_nan_predictions",
			"management.deploymentInfo",
			"model_deployments.None",
			"model_deployments.deployment_sharing",
			"model_deployments.model_replacement",
			"prediction_request.None",
			"prediction_request.failed",
			"model_deployments.model_replacement_lifecycle",
			"model_deployments.model_replacement_started",
			"model_deployments.model_replacement_succeeded",
			"model_deployments.model_replacement_failed",
			"model_deployments.model_replacement_validation_warning",
			"model_deployments.deployment_creation",
			"model_deployments.deployment_deletion",
			"model_deployments.service_health_yellow_from_green",
			"model_deployments.service_health_yellow_from_red",
			"model_deployments.service_health_red",
			"model_deployments.data_drift_yellow_from_green",
			"model_deployments.data_drift_yellow_from_red",
			"model_deployments.data_drift_red",
			"model_deployments.accuracy_yellow_from_green",
			"model_deployments.accuracy_yellow_from_red",
			"model_deployments.accuracy_red",
			"model_deployments.health.fairness_health.green_to_yellow",
			"model_deployments.health.fairness_health.red_to_yellow",
			"model_deployments.health.fairness_health.red",
			"model_deployments.health.custom_metrics_health.green_to_yellow",
			"model_deployments.health.custom_metrics_health.red_to_yellow",
			"model_deployments.health.custom_metrics_health.red",
			"model_deployments.health.base.green",
			"model_deployments.service_health_green",
			"model_deployments.data_drift_green",
			"model_deployments.accuracy_green",
			"model_deployments.health.fairness_health.green",
			"model_deployments.health.custom_metrics_health.green",
			"model_deployments.retraining_policy_run_started",
			"model_deployments.retraining_policy_run_succeeded",
			"model_deployments.retraining_policy_run_failed",
			"model_deployments.challenger_scoring_success",
			"model_deployments.challenger_scoring_data_warning",
			"model_deployments.challenger_scoring_failure",
			"model_deployments.challenger_scoring_started",
			"model_deployments.challenger_model_validation_warning",
			"model_deployments.challenger_model_created",
			"model_deployments.challenger_model_deleted",
			"model_deployments.actuals_upload_failed",
			"model_deployments.actuals_upload_warning",
			"model_deployments.training_data_baseline_calculation_started",
			"model_deployments.training_data_baseline_calculation_completed",
			"model_deployments.training_data_baseline_failed",
			"model_deployments.custom_model_deployment_creation_started",
			"model_deployments.custom_model_deployment_creation_completed",
			"model_deployments.custom_model_deployment_creation_failed",
			"model_deployments.deployment_prediction_explanations_preview_job_submitted",
			"model_deployments.deployment_prediction_explanations_preview_job_completed",
			"model_deployments.deployment_prediction_explanations_preview_job_failed",
			"model_deployments.custom_model_deployment_activated",
			"model_deployments.custom_model_deployment_activation_failed",
			"model_deployments.custom_model_deployment_deactivated",
			"model_deployments.custom_model_deployment_deactivation_failed",
			"model_deployments.prediction_processing_rate_limit_reached",
			"model_deployments.prediction_data_processing_rate_limit_reached",
			"model_deployments.prediction_data_processing_rate_limit_warning",
			"model_deployments.actuals_processing_rate_limit_reached",
			"model_deployments.actuals_processing_rate_limit_warning",
			"model_deployments.deployment_monitoring_data_cleared",
			"model_deployments.deployment_launch_started",
			"model_deployments.deployment_launch_succeeded",
			"model_deployments.deployment_launch_failed",
			"model_deployments.deployment_shutdown_started",
			"model_deployments.deployment_shutdown_succeeded",
			"model_deployments.deployment_shutdown_failed",
			"model_deployments.endpoint_update_started",
			"model_deployments.endpoint_update_succeeded",
			"model_deployments.endpoint_update_failed",
			"model_deployments.management_agent_service_health_green",
			"model_deployments.management_agent_service_health_yellow",
			"model_deployments.management_agent_service_health_red",
			"model_deployments.management_agent_service_health_unknown",
			"model_deployments.predictions_missing_association_id",
			"model_deployments.prediction_result_rows_cleand_up",
			"model_deployments.batch_deleted",
			"model_deployments.batch_creation_limit_reached",
			"model_deployments.batch_creation_limit_exceeded",
			"model_deployments.batch_not_found",
			"model_deployments.predictions_encountered_for_locked_batch",
			"model_deployments.predictions_encountered_for_deleted_batch",
			"model_deployments.scheduled_report_generated",
			"model_deployments.predictions_timeliness_health_red",
			"model_deployments.actuals_timeliness_health_red",
			"model_deployments.service_health_still_red",
			"model_deployments.data_drift_still_red",
			"model_deployments.accuracy_still_red",
			"model_deployments.health.fairness_health.still_red",
			"model_deployments.health.custom_metrics_health.still_red",
			"model_deployments.predictions_timeliness_health_still_red",
			"model_deployments.actuals_timeliness_health_still_red",
			"model_deployments.service_health_still_yellow",
			"model_deployments.data_drift_still_yellow",
			"model_deployments.accuracy_still_yellow",
			"model_deployments.health.fairness_health.still_yellow",
			"model_deployments.health.custom_metrics_health.still_yellow",
			"model_deployments.deployment_inference_server_creation_started",
			"model_deployments.deployment_inference_server_creation_failed",
			"model_deployments.deployment_inference_server_creation_completed",
			"model_deployments.deployment_inference_server_deletion",
			"model_deployments.deployment_inference_server_idle_stopped",
			"entity_notification_policy_template.shared",
			"notification_channel_template.shared",
			"project.created",
			"project.deleted",
			"project.shared",
			"autopilot.complete",
			"autopilot.started",
			"autostart.failure",
			"perma_delete_project.success",
			"perma_delete_project.failure",
			"users_delete.preview_started",
			"users_delete.preview_completed",
			"users_delete.preview_failed",
			"users_delete.started",
			"users_delete.completed",
			"users_delete.failed",
			"application.created",
			"application.shared",
			"model_version.added",
			"model_version.stage_transition_from_registered_to_development",
			"model_version.stage_transition_from_registered_to_staging",
			"model_version.stage_transition_from_registered_to_production",
			"model_version.stage_transition_from_registered_to_archived",
			"model_version.stage_transition_from_development_to_registered",
			"model_version.stage_transition_from_development_to_staging",
			"model_version.stage_transition_from_development_to_production",
			"model_version.stage_transition_from_development_to_archived",
			"model_version.stage_transition_from_staging_to_registered",
			"model_version.stage_transition_from_staging_to_development",
			"model_version.stage_transition_from_staging_to_production",
			"model_version.stage_transition_from_staging_to_archived",
			"model_version.stage_transition_from_production_to_registered",
			"model_version.stage_transition_from_production_to_development",
			"model_version.stage_transition_from_production_to_staging",
			"model_version.stage_transition_from_production_to_archived",
			"model_version.stage_transition_from_archived_to_registered",
			"model_version.stage_transition_from_archived_to_development",
			"model_version.stage_transition_from_archived_to_production",
			"model_version.stage_transition_from_archived_to_staging",
			"batch_predictions.success",
			"batch_predictions.failed",
			"batch_predictions.scheduler.auto_disabled",
			"change_request.cancelled",
			"change_request.created",
			"change_request.deployment_approval_requested",
			"change_request.resolved",
			"change_request.proposed_changes_updated",
			"change_request.pending",
			"change_request.commenting_review_added",
			"change_request.approving_review_added",
			"change_request.changes_requesting_review_added",
			"prediction_explanations_computation.None",
			"prediction_explanations_computation.prediction_explanations_preview_job_submitted",
			"prediction_explanations_computation.prediction_explanations_preview_job_completed",
			"prediction_explanations_computation.prediction_explanations_preview_job_failed",
			"monitoring.rate_limit_enforced",
			"notebook_schedule.created",
			"notebook_schedule.failure",
			"notebook_schedule.completed",
			"abstract",
			"moderation.metric.creation_error",
			"moderation.metric.reporting_error",
			"moderation.model.moderation_started",
			"moderation.model.moderation_completed",
			"moderation.model.pre_score_phase_started",
			"moderation.model.pre_score_phase_completed",
			"moderation.model.post_score_phase_started",
			"moderation.model.post_score_phase_completed",
			"moderation.model.config_error",
			"moderation.model.runtime_error",
			"moderation.model.scoring_started",
			"moderation.model.scoring_completed",
			"moderation.model.scoring_error",
		),
	}
}

func CustomMetricAggregationTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"average",
			"categorical",
			"gauge",
			"sum",
		),
	}
}

func BatchPredictionJobIntakeTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"localFile",
			"s3",
			"azure",
			"gcp",
			"dataset",
			"jdbc",
			"snowflake",
			"synapse",
			"bigquery",
			"datasphere",
		),
	}
}

func BatchPredictionJobOutputTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"localFile",
			"s3",
			"azure",
			"gcp",
			"jdbc",
			"snowflake",
			"synapse",
			"bigquery",
			"datasphere",
		),
	}
}

func BatchPredictionJobTimeSeriesTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"forecast",
			"historical",
		),
	}
}

func BatchPredictionJobExplanationAlgorithmValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"shap",
			"xemp",
		),
	}
}
