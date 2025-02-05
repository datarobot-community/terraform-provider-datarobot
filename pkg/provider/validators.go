package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
