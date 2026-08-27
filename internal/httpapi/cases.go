package httpapi

import (
	"net/http"

	"github.com/benzhi/oral-history-release/internal/application"
)

func (a *API) ListCasesHandler(w http.ResponseWriter, _ *http.Request) {
	items, err := a.service.ListCases()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) CreateCaseHandler(w http.ResponseWriter, r *http.Request) {
	var command application.CreateCaseCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.CreateCase(command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusCreated, body)
}

func (a *API) GetCaseHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	detail, err := a.service.GetCase(caseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (a *API) FreezePreviewHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	preview, err := a.service.GetFreezePreview(caseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (a *API) AddConsentHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.AddConsentCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.AddConsent(caseID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) WithdrawConsentHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	consentID, err := requestID(r, "consentID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.WithdrawConsentCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.WithdrawConsentContext(r.Context(), caseID, consentID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) SubmitTranscriptHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.SubmitTranscriptCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.SubmitTranscript(caseID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}
