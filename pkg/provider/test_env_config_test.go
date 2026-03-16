package provider

import "os"

// Centralized test environment configuration.
//
// All test environment IDs are loaded from environment variables with fallback defaults.
// The .env file (loaded in provider_test.go init()) is the recommended way to override
// these values for local development. CI/CD systems can set the env vars directly.
//
// To override a value, add it to the .env file at the project root:
//
//	DR_TEST_GENAI_BASE_ENV_ID=67ab469cecdca772287de644
//	DR_TEST_STREAMLIT_BASE_ENV_ID=6542cd582a9d3d51bf4ac71e
//	DR_TEST_CUSTOM_JOB_ENV_ID=66d07fae0513a1edf18595bb
//	DR_TEST_APP_SOURCE_BASE_ENV_VERSION_ID=668548c1b8e086572a96fbf5
//	DR_TEST_CUSTOM_APP_ENV_ID=67987589391fe8fa0a2275b8
//	DR_TEST_CUSTOM_APP_ENV_ID_2=67987b1a90dbd55389b699c2
//	DR_TEST_SLACKBOT_TEMPLATE_ID=67126757e7819551baceb22b
//	DR_TEST_QA_TEMPLATE_ID=670fb324bf9bbb1081114333
//	DR_TEST_STREAMLIT_TEMPLATE_ID=671267597665e0b33f7acdb7
//	DR_TEST_FLASK_TEMPLATE_ID=67126750e8342440587acd74
//	DR_TEST_LEADERBOARD_MODEL_ID=673b722dfd279fd86944d088
//	DR_TEST_LEADERBOARD_MODEL_ID_2=673b6fd8e060b90658aebe66
//	DR_TEST_LEADERBOARD_STAGING_MODEL_ID=673b75ec97f1021bbfb61d3b
//	DR_TEST_LEADERBOARD_STAGING_MODEL_ID_2=673b75ec97f1021bbfb61d34
//	DR_TEST_DR_GROUP_ENTITY_ID=6036d237608973bf082aba1e
var (
	// Base environment IDs.

	// testGenAIBaseEnvID is the [GenAI] Python 3.11 base environment.
	// Used for custom models (Binary, Regression, TextGeneration, MCP, etc.).
	testGenAIBaseEnvID string

	// testStreamlitBaseEnvID is the [Experimental] Python 3.9 Streamlit base environment.
	// Used for application sources and custom applications.
	testStreamlitBaseEnvID string

	// testCustomJobEnvID is the base environment for custom jobs and metric jobs.
	testCustomJobEnvID string

	// Base environment version IDs.

	// testAppSourceBaseEnvVersionID is a specific version of the Streamlit base environment.
	// Used in application source and application source from template tests.
	testAppSourceBaseEnvVersionID string

	// Custom application execution environment IDs.

	// testCustomAppEnvID is an execution environment for custom applications from environment.
	testCustomAppEnvID string

	// testCustomAppEnvID2 is a second execution environment for custom applications from environment.
	testCustomAppEnvID2 string

	// Template IDs.

	// testSlackbotTemplateID is the Slackbot application template.
	testSlackbotTemplateID string

	// testQATemplateID is the Q&A application template.
	testQATemplateID string

	// testStreamlitTemplateID is the Streamlit application template.
	testStreamlitTemplateID string

	// testFlaskTemplateID is the Flask application template.
	testFlaskTemplateID string

	// Leaderboard / registered model IDs.

	// testLeaderboardModelID is a model from a project leaderboard (production).
	testLeaderboardModelID string

	// testLeaderboardModelID2 is a second model from a project leaderboard (production).
	testLeaderboardModelID2 string

	// testLeaderboardStagingModelID is a model from a project leaderboard (staging).
	testLeaderboardStagingModelID string

	// testLeaderboardStagingModelID2 is a second model from a project leaderboard (staging).
	testLeaderboardStagingModelID2 string

	// Other entity IDs.

	// testDRGroupEntityID is a DataRobot group entity ID for notification channel tests.
	testDRGroupEntityID string
)

func init() {
	// Base environments
	testGenAIBaseEnvID = getTestEnvOrDefault("DR_TEST_GENAI_BASE_ENV_ID", "67ab469cecdca772287de644")
	testStreamlitBaseEnvID = getTestEnvOrDefault("DR_TEST_STREAMLIT_BASE_ENV_ID", "6542cd582a9d3d51bf4ac71e")
	testCustomJobEnvID = getTestEnvOrDefault("DR_TEST_CUSTOM_JOB_ENV_ID", "66d07fae0513a1edf18595bb")

	// Base environment versions
	testAppSourceBaseEnvVersionID = getTestEnvOrDefault("DR_TEST_APP_SOURCE_BASE_ENV_VERSION_ID", "668548c1b8e086572a96fbf5")

	// Custom application execution environments
	testCustomAppEnvID = getTestEnvOrDefault("DR_TEST_CUSTOM_APP_ENV_ID", "67987589391fe8fa0a2275b8")
	testCustomAppEnvID2 = getTestEnvOrDefault("DR_TEST_CUSTOM_APP_ENV_ID_2", "67987b1a90dbd55389b699c2")

	// Templates
	testSlackbotTemplateID = getTestEnvOrDefault("DR_TEST_SLACKBOT_TEMPLATE_ID", "67126757e7819551baceb22b")
	testQATemplateID = getTestEnvOrDefault("DR_TEST_QA_TEMPLATE_ID", "670fb324bf9bbb1081114333")
	testStreamlitTemplateID = getTestEnvOrDefault("DR_TEST_STREAMLIT_TEMPLATE_ID", "671267597665e0b33f7acdb7")
	testFlaskTemplateID = getTestEnvOrDefault("DR_TEST_FLASK_TEMPLATE_ID", "67126750e8342440587acd74")

	// Leaderboard models
	testLeaderboardModelID = getTestEnvOrDefault("DR_TEST_LEADERBOARD_MODEL_ID", "673b722dfd279fd86944d088")
	testLeaderboardModelID2 = getTestEnvOrDefault("DR_TEST_LEADERBOARD_MODEL_ID_2", "673b6fd8e060b90658aebe66")
	testLeaderboardStagingModelID = getTestEnvOrDefault("DR_TEST_LEADERBOARD_STAGING_MODEL_ID", "673b75ec97f1021bbfb61d3b")
	testLeaderboardStagingModelID2 = getTestEnvOrDefault("DR_TEST_LEADERBOARD_STAGING_MODEL_ID_2", "673b75ec97f1021bbfb61d34")

	// Other entities
	testDRGroupEntityID = getTestEnvOrDefault("DR_TEST_DR_GROUP_ENTITY_ID", "6036d237608973bf082aba1e")
}

func getTestEnvOrDefault(envVar, defaultValue string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}
