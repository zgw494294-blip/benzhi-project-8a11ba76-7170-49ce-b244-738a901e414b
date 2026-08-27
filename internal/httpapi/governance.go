package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/benzhi/oral-history-release/internal/application"
)

func (a *API) AddFindingHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.AddFindingCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	var body json.RawMessage
	if len(command.Items) > 0 || len(command.Findings) > 0 {
		batch := application.BatchFindingsCommand{CommandMeta: command.CommandMeta, Items: command.Items, Findings: command.Findings}
		body, err = a.service.AddFindingsBatch(caseID, batch)
	} else {
		body, err = a.service.AddFinding(caseID, command)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) ReviseFindingHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	findingID, err := requestID(r, "findingID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.ReviseFindingCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.ReviseFinding(caseID, findingID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) RaiseObjectionHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.RaiseObjectionCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.RaiseObjection(caseID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}

func (a *API) CloseObjectionHandler(w http.ResponseWriter, r *http.Request) {
	caseID, err := requestID(r, "caseID")
	if err != nil {
		writeError(w, err)
		return
	}
	objectionID, err := requestID(r, "objectionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var command application.CloseObjectionCommand
	if err := decodeJSON(w, r, &command); err != nil {
		writeError(w, err)
		return
	}
	body, err := a.service.CloseObjection(caseID, objectionID, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, http.StatusOK, body)
}
