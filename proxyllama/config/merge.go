package config

func MergeWithDefaultConfig(userConfig *UserConfig) {
	base := GetConfig()

	// Fill in nil fields in userConfig from base config
	if userConfig.ModelProfiles == nil {
		userConfig.ModelProfiles = &base.ModelProfiles
	}
	if userConfig.Summarization == nil {
		userConfig.Summarization = &base.Summarization
	}
	if userConfig.Retrieval == nil {
		userConfig.Retrieval = &base.Retrieval
	}
	if userConfig.WebSearch == nil {
		userConfig.WebSearch = &base.WebSearch
	}
	if userConfig.Preferences == nil {
		userConfig.Preferences = &base.Preferences
	}
}
