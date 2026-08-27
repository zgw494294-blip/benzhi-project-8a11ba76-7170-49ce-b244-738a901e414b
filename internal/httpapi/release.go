package httpapi

import (
	"net/http"

	"github.com/benzhi/oral-history-release/internal/application"
)

func (a *API) FreezeHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.FreezeCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.Freeze(caseID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.AuthorizeCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.Authorize(caseID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) VerifyCredentialHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	credentialID, err := requestID(r, "credentialID")
	if err != nil {
		writeError(w, err)
		return
	}
	verification, err := a.service.VerifyCredential(caseID, credentialID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, verification)
}
