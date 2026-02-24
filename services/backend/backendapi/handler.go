package backendapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/generic/httperrors"
	"github.com/google/uuid"
)

func NewHandler(svc backend.ConversationService) http.Handler {
	h := &httpHandler{
		svc: svc,
	}
	h.init()
	return h
}

type httpHandler struct {
	http.ServeMux
	svc backend.ConversationService
}

func (h *httpHandler) init() {
	h.HandleFunc("GET /slack", h.completeSlackAuthentication)
	h.HandleFunc("POST /reply", h.sendReply)
}

func (h *httpHandler) completeSlackAuthentication(w http.ResponseWriter, r *http.Request) {
	type request struct{}
	type response struct{}

	code := r.URL.Query().Get("code")

	ApiHandlerFunc(func(ctx context.Context, x request) (response, error) {
		err := h.svc.CompleteSlackIntegration(ctx, backend.CompleteSlackIntegrationCommand{
			BusinessID: uuid.New().String(),
			Code:       code,
		})
		if err != nil {
			slog.Error("error in complete slack authentication", "err", err)
			return response{}, err
		}
		return response{}, nil
	})(w, r)
}

func (h *httpHandler) sendReply(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ConversationID string `json:"conversation_id"`
		Message        string `json:"message"`
	}
	type response struct{}

	ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		err := h.svc.SendReply(ctx, backend.SendReplyCommand{
			ConversationID: req.ConversationID,
			Message:        req.Message,
		})
		if err != nil {
			slog.Error("error sending reply", "err", err)
			return response{}, err
		}
		return response{}, nil
	})(w, r)
}

func ApiHandlerFunc[X any, Y any](api func(
	context.Context, X) (Y, error)) func(http.ResponseWriter, *http.Request) {
	const RequestIDHeader = "x-request-id"

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		request := new(X)
		bodyBytes, err := io.ReadAll(r.Body)

		if err := json.Unmarshal(bodyBytes, request); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		res, err := api(ctx, *request)
		if err != nil {
			slog.Error("error in api handler", "path", r.URL, "request", request, "err", err)
			var httpError = httperrors.From(err)
			w.WriteHeader(httpError.HttpStatus)
			_ = json.NewEncoder(w).Encode(httpError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(res)
	}
}
