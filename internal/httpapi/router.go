package httpapi

import (
	"net/http"
	"time"

	"github.com/benzhi/oral-history-release/internal/application"
	"github.com/benzhi/oral-history-release/internal/webassets"
)

type API struct {
	service *application.Service
}

func New(service *application.Service) *API { return &API{service: service} }

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", webassets.WorkbenchHandler())
	mux.Handle("GET /assets/", webassets.StaticHandler())
	mux.HandleFunc("GET /healthz", a.HealthHandler)
	mux.HandleFunc("GET /api/v1/cases", a.ListCasesHandler)
	mux.HandleFunc("POST /api/v1/cases", a.CreateCaseHandler)
	mux.HandleFunc("GET /api/v1/cases/{caseID}", a.GetCaseHandler)
	mux.HandleFunc("GET /api/v1/cases/{caseID}/freeze-preview", a.FreezePreviewHandler)
	mux.HandleFunc("GET /api/v1/cases/{caseID}/freeze/preview", a.FreezePreviewHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/consents", a.AddConsentHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/consents/{consentID}/withdraw", a.WithdrawConsentHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/transcripts", a.SubmitTranscriptHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/findings", a.AddFindingHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/findings/batch", a.AddFindingHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/findings/{findingID}/revise", a.ReviseFindingHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/objections", a.RaiseObjectionHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/objections/{objectionID}/close", a.CloseObjectionHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/freeze", a.FreezeHandler)
	mux.HandleFunc("POST /api/v1/cases/{caseID}/authorize", a.AuthorizeHandler)
	mux.HandleFunc("GET /api/v1/cases/{caseID}/credentials/{credentialID}/verify", a.VerifyCredentialHandler)
	return securityHeaders(requestLogGuard(mux))
}

func (a *API) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func requestLogGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && len(r.URL.Path) > 512 {
			writeJSON(w, http.StatusRequestURITooLong, errorBody{Code: "uri_too_long", Message: "请求路径过长"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
