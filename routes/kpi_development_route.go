package routes

import (
	"net/http"

	"kpiproject/handlers"
	"kpiproject/middlewares"
)

func SetupKPIRoutes(kpiHandler *handlers.KPIHandler, jwtSecret string) *http.ServeMux {
	mux := http.NewServeMux()

	// Apply JWT middleware to all KPI routes
	jwtMiddleware := middlewares.JWTMiddleware(jwtSecret)

	// KPI Development routes with JWT protection
	mux.Handle("POST /api/kpi", jwtMiddleware(http.HandlerFunc(kpiHandler.CreateKPI)))
	mux.Handle("GET /api/kpi", jwtMiddleware(http.HandlerFunc(kpiHandler.GetAllKPIs)))
	mux.Handle("GET /api/kpi/{id}", jwtMiddleware(http.HandlerFunc(kpiHandler.GetKPIByID)))
	mux.Handle("PUT /api/kpi/{id}", jwtMiddleware(http.HandlerFunc(kpiHandler.UpdateKPI)))
	mux.Handle("DELETE /api/kpi/{id}", jwtMiddleware(http.HandlerFunc(kpiHandler.DeleteKPI)))
	// File attachment routes
	mux.Handle("POST /api/kpi/{id}/attachments", jwtMiddleware(http.HandlerFunc(kpiHandler.UploadAttachment)))
	mux.Handle("GET /api/kpi/attachments/{fileId}/download", jwtMiddleware(http.HandlerFunc(kpiHandler.DownloadAttachment)))
	mux.Handle("DELETE /api/kpi/{id}/attachments/{fileId}", jwtMiddleware(http.HandlerFunc(kpiHandler.DeleteAttachment)))
	// File transfer with transaction
	mux.Handle("POST /api/kpi/attachments/transfer", jwtMiddleware(http.HandlerFunc(kpiHandler.TransferAttachment)))
	// Analytics routes
	mux.Handle("GET /api/kpi/analytics/performance", jwtMiddleware(http.HandlerFunc(kpiHandler.GetKPIPerformanceStats)))

	return mux
}
