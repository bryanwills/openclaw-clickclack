package httpapi

import "net/http"

func (s *Server) requestMagicLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	link, err := s.store.CreateMagicLink(r.Context(), body.Email, body.DisplayName)
	writeResultStatus(w, http.StatusCreated, map[string]any{"magic_link": link, "token": link.Token}, err)
}

func (s *Server) consumeMagicLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, session, err := s.store.ConsumeMagicLink(r.Context(), body.Token)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	setSessionCookie(w, session)
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "session": session, "token": session.Token})
}
