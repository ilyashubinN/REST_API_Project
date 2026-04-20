package redirect

import (
	"errors"
	"log/slog"
	"net/http"

	resp "go-postgres-test/internal/lib/api/response"
	"go-postgres-test/internal/lib/logger/sl"
	"go-postgres-test/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

//go:generate go run github.com/vektra/mockery/v2@latest --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		reqLog := log.With( // CHANGED
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			reqLog.Info("alias is empty") // CHANGED

			w.WriteHeader(http.StatusBadRequest) // CHANGED
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		resURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			reqLog.Info("url not found", slog.String("alias", alias)) // CHANGED

			w.WriteHeader(http.StatusNotFound) // CHANGED
			render.JSON(w, r, resp.Error("not found"))
			return
		}

		if err != nil {
			reqLog.Error("failed to get url", sl.Err(err)) // CHANGED

			w.WriteHeader(http.StatusInternalServerError) // CHANGED
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		reqLog.Info( // CHANGED
			"got url",
			slog.String("alias", alias),
			slog.String("url", resURL),
		)

		http.Redirect(w, r, resURL, http.StatusFound)
	}
}
