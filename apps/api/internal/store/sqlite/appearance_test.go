package sqlite

import (
	"context"
	"testing"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func TestAppearancePreferencesLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, store.CreateUserInput{
		DisplayName: "Appearance User",
		Email:       "appearance-sqlite@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	preferences, err := st.GetAppearancePreferences(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preferences != nil {
		t.Fatalf("expected missing preferences, got %#v", preferences)
	}

	if preferences, err = st.UpdateAppearancePreferences(ctx, store.UpdateAppearancePreferencesInput{
		UserID: user.ID,
	}); err != nil {
		t.Fatal(err)
	} else if preferences != nil {
		t.Fatalf("empty patch created a row: %#v", preferences)
	}

	system := "system"
	moss := "moss"
	compact := "compact"
	preferences, err = st.UpdateAppearancePreferences(ctx, store.UpdateAppearancePreferencesInput{
		UserID: user.ID,
		Patch: store.AppearancePreferencesPatch{
			ColorMode:  &system,
			BoardTheme: &moss,
			Density:    &compact,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preferences == nil || preferences.ColorMode != "" || preferences.BoardTheme != "moss" || preferences.MessageLayout != "" || preferences.Density != "compact" {
		t.Fatalf("unexpected initial preferences: %#v", preferences)
	}

	signal := "signal"
	outlined := "outlined"
	preferences, err = st.UpdateAppearancePreferences(ctx, store.UpdateAppearancePreferencesInput{
		UserID: user.ID,
		Patch: store.AppearancePreferencesPatch{
			BoardTheme:    &signal,
			MessageLayout: &outlined,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preferences == nil || preferences.ColorMode != "" || preferences.BoardTheme != "" || preferences.MessageLayout != "outlined" || preferences.Density != "compact" {
		t.Fatalf("partial update replaced unrelated preferences: %#v", preferences)
	}

	invalid := "sepia"
	if _, err := st.UpdateAppearancePreferences(ctx, store.UpdateAppearancePreferencesInput{
		UserID: user.ID,
		Patch:  store.AppearancePreferencesPatch{ColorMode: &invalid},
	}); err == nil {
		t.Fatal("expected invalid color mode to fail")
	}
	preferences, err = st.GetAppearancePreferences(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preferences == nil || preferences.MessageLayout != "outlined" || preferences.Density != "compact" {
		t.Fatalf("invalid update changed preferences: %#v", preferences)
	}

	if _, err := st.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, user.ID); err != nil {
		t.Fatal(err)
	}
	preferences, err = st.GetAppearancePreferences(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preferences != nil {
		t.Fatalf("user deletion did not cascade preferences: %#v", preferences)
	}
}
