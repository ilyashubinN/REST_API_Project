package save_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-postgres-test/internal/http-server/handlers/url/save"
	resp "go-postgres-test/internal/lib/api/response"
	"go-postgres-test/internal/storage"
)

type urlSaverMock struct {
	mockFunc func(urlToSave string, alias string) (int64, error)
}

func (m *urlSaverMock) SaveURL(urlToSave string, alias string) (int64, error) {
	return m.mockFunc(urlToSave, alias)
}

func TestSaveHandler(t *testing.T) {
	type wantResp struct {
		status string
		error  string
		alias  string
	}

	cases := []struct {
		name      string
		alias     string
		url       string
		body      io.Reader
		mockError error
		want      wantResp
	}{
		{
			name:  "success",
			alias: "test_alias",
			url:   "https://google.com",
			want: wantResp{
				status: resp.StatusOK,
				error:  "",
				alias:  "test_alias",
			},
		},
		{
			name:  "empty alias",
			alias: "",
			url:   "https://google.com",
			want: wantResp{
				status: resp.StatusOK,
				error:  "",
				// если в handler генерируется alias автоматически,
				// здесь можно не проверять конкретное значение
			},
		},
		{
			name:  "invalid url",
			alias: "test_alias",
			url:   "not-a-url",
			want: wantResp{
				status: resp.StatusError,
				error:  "field URL is not a valid URL",
			},
		},
		{
			name:      "url already exists",
			alias:     "test_alias",
			url:       "https://google.com",
			mockError: storage.ErrURLExists,
			want: wantResp{
				status: resp.StatusError,
				error:  "url already exists",
			},
		},
		{
			name: "invalid json",
			body: bytes.NewBufferString(`{"url":`),
			want: wantResp{
				status: resp.StatusError,
				error:  "failed to decode request",
			},
		},
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saverMock := &urlSaverMock{
				mockFunc: func(urlToSave string, alias string) (int64, error) {
					if tc.mockError != nil {
						return 0, tc.mockError
					}
					return 1, nil
				},
			}

			var body io.Reader
			if tc.body != nil {
				body = tc.body
			} else {
				reqBody := save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}

				data, err := json.Marshal(reqBody)
				if err != nil {
					t.Fatalf("failed to marshal request body: %v", err)
				}

				body = bytes.NewBuffer(data)
			}

			req := httptest.NewRequest(http.MethodPost, "/url", body)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler := save.New(log, saverMock)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, rr.Code)
			}

			var got save.Response
			err := json.Unmarshal(rr.Body.Bytes(), &got)
			if err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}

			if got.Status != tc.want.status {
				t.Fatalf("expected status %q, got %q", tc.want.status, got.Status)
			}

			if got.Error != tc.want.error {
				t.Fatalf("expected error %q, got %q", tc.want.error, got.Error)
			}

			// alias проверяем только если ожидаем конкретное значение
			if tc.want.alias != "" && got.Alias != tc.want.alias {
				t.Fatalf("expected alias %q, got %q", tc.want.alias, got.Alias)
			}
		})
	}
}
