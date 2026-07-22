package store

import "errors"

func AppearancePreferencesPatchEmpty(patch AppearancePreferencesPatch) bool {
	return patch.ColorMode == nil &&
		patch.BoardTheme == nil &&
		patch.MessageLayout == nil &&
		patch.Density == nil
}

func NormalizeAppearancePreferencesPatch(input AppearancePreferencesPatch) (AppearancePreferencesPatch, error) {
	colorMode, err := normalizeAppearancePreference(input.ColorMode, map[string]string{
		"":       "",
		"system": "",
		"light":  "light",
		"dark":   "dark",
	}, "color_mode")
	if err != nil {
		return AppearancePreferencesPatch{}, err
	}
	boardTheme, err := normalizeAppearancePreference(input.BoardTheme, map[string]string{
		"":       "",
		"signal": "",
		"ember":  "ember",
		"moss":   "moss",
		"iris":   "iris",
	}, "board_theme")
	if err != nil {
		return AppearancePreferencesPatch{}, err
	}
	messageLayout, err := normalizeAppearancePreference(input.MessageLayout, map[string]string{
		"":         "",
		"standard": "",
		"outlined": "outlined",
	}, "message_layout")
	if err != nil {
		return AppearancePreferencesPatch{}, err
	}
	density, err := normalizeAppearancePreference(input.Density, map[string]string{
		"":            "",
		"comfortable": "",
		"compact":     "compact",
	}, "density")
	if err != nil {
		return AppearancePreferencesPatch{}, err
	}
	return AppearancePreferencesPatch{
		ColorMode:     colorMode,
		BoardTheme:    boardTheme,
		MessageLayout: messageLayout,
		Density:       density,
	}, nil
}

func normalizeAppearancePreference(value *string, allowed map[string]string, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized, ok := allowed[*value]
	if !ok {
		return nil, errors.New(field + " is invalid")
	}
	return &normalized, nil
}
