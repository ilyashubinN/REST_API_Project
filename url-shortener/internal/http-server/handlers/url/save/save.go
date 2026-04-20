package save

import (
	"errors"
	"log/slog"
	"net/http"

	resp "go-postgres-test/internal/lib/api/response"
	"go-postgres-test/internal/lib/logger/sl"
	"go-postgres-test/internal/lib/random"
	"go-postgres-test/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

const aliasLength = 4

//go:generate go run github.com/vektra/mockery/v2@latest --name=URLSaver

type URLSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		reqLog := log.With( // CHANGED: не перезаписываем входной log, а создаём отдельный reqLog
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil { // CHANGED: if err := ...; err != nil
			reqLog.Error("failed to decode request body", sl.Err(err))

			w.WriteHeader(http.StatusBadRequest) // CHANGED: явный статус 400
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}

		reqLog.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			reqLog.Error("invalid request", sl.Err(err))

			w.WriteHeader(http.StatusBadRequest) // CHANGED: явный статус 400
			render.JSON(w, r, resp.ValidationError(validateErr))
			return
		}

		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(aliasLength)
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrURLExists) {
			reqLog.Info("url already exists", slog.String("url", req.URL))

			w.WriteHeader(http.StatusConflict) // CHANGED: явный статус 409
			render.JSON(w, r, resp.Error("url already exists"))
			return
		}

		if err != nil {
			reqLog.Error("failed to add url", sl.Err(err))

			w.WriteHeader(http.StatusInternalServerError) // CHANGED: явный статус 500
			render.JSON(w, r, resp.Error("failed to add url"))
			return
		}

		reqLog.Info("url added",
			slog.Int64("id", id),
			slog.String("alias", alias), // CHANGED: добавил alias в лог
		)

		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	w.WriteHeader(http.StatusOK) // CHANGED: явный статус 200
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
