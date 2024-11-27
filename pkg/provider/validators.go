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
