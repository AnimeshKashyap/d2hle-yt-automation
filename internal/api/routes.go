package api

type RouteDoc struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        bool   `json:"auth_required"`
	Summary     string `json:"summary"`
	Implemented string `json:"implemented"` // "yes" | "partial" | "no"
}

func routeCatalog() []RouteDoc {
	return []RouteDoc{
		{Method: "GET", Path: "/v1/docs", Auth: false, Summary: "List all HTTP API routes", Implemented: "yes"},
		{Method: "GET", Path: "/v1/health", Auth: false, Summary: "Liveness probe", Implemented: "yes"},
		{Method: "GET", Path: "/v1/ready", Auth: false, Summary: "Whether an upload execution slot is free", Implemented: "yes"},
		{Method: "POST", Path: "/v1/profiles", Auth: true, Summary: "Register a new profile", Implemented: "yes"},
		{Method: "GET", Path: "/v1/profiles", Auth: false, Summary: "List profiles and session_status", Implemented: "yes"},
		{Method: "GET", Path: "/v1/profiles/{profile_id}", Auth: false, Summary: "Get profile details", Implemented: "yes"},
		{Method: "GET", Path: "/v1/profiles/{profile_id}/status", Auth: false, Summary: "Profile session status (alias)", Implemented: "yes"},
		{Method: "POST", Path: "/v1/profiles/{profile_id}/tokens", Auth: true, Summary: "Issue service API bearer token", Implemented: "yes"},
		{Method: "DELETE", Path: "/v1/profiles/{profile_id}/tokens/{token_id}", Auth: true, Summary: "Revoke service API token", Implemented: "yes"},
		{Method: "POST", Path: "/v1/profiles/{profile_id}/session/check", Auth: true, Summary: "Studio probe; updates session_status", Implemented: "yes"},
		{Method: "POST", Path: "/v1/profiles/{profile_id}/session/refresh", Auth: true, Summary: "Headed session warm-up (cookie auth); sets last_refreshed_at", Implemented: "yes"},
		{Method: "POST", Path: "/v1/profiles/{profile_id}/login", Auth: true, Summary: "Start login (CLI or optional headed interactive)", Implemented: "yes"},
		{Method: "POST", Path: "/v1/jobs", Auth: true, Summary: "Submit upload job and start Studio automation", Implemented: "yes"},
		{Method: "GET", Path: "/v1/jobs/current", Auth: false, Summary: "Currently running job, if any", Implemented: "yes"},
		{Method: "GET", Path: "/v1/jobs/{job_id}", Auth: false, Summary: "Get job by id", Implemented: "yes"},
	}
}
