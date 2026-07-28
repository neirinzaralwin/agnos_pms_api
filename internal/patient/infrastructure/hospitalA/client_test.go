package hospitalA_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/infrastructure/hospitalA"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

func TestSearchByID_OK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/patient/search/1100700123456", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"first_name_en":"Somchai","last_name_en":"Jaidee",
			"national_id":"1100700123456","gender":"M","date_of_birth":"1990-01-15"
		}`))
	}))
	t.Cleanup(srv.Close)

	client, operationError := hospitalA.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, operationError)

	patient, operationError := client.SearchByID(context.Background(), "1100700123456")
	require.NoError(t, operationError)
	require.NotNil(t, patient.NationalID)
	require.Equal(t, "1100700123456", *patient.NationalID)
	require.Equal(t, "M", *patient.Gender)
}

func TestSearchByID_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client, operationError := hospitalA.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, operationError)

	_, operationError = client.SearchByID(context.Background(), "MISSING")
	require.ErrorIs(t, operationError, platform.ErrNotFound)
}

func TestSearchByID_Upstream500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	client, operationError := hospitalA.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, operationError)

	_, operationError = client.SearchByID(context.Background(), "X")
	require.ErrorIs(t, operationError, platform.ErrUpstream)
}

func TestSearchByID_BadJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{bad`))
	}))
	t.Cleanup(srv.Close)

	client, operationError := hospitalA.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, operationError)

	_, operationError = client.SearchByID(context.Background(), "X")
	require.ErrorIs(t, operationError, platform.ErrUpstream)
}

func TestSearchByID_Timeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client, operationError := hospitalA.New(srv.URL, 50*time.Millisecond, nil)
	require.NoError(t, operationError)

	_, operationError = client.SearchByID(context.Background(), "X")
	require.ErrorIs(t, operationError, platform.ErrUpstream)
}

func TestSearchByID_InvalidID(t *testing.T) {
	t.Parallel()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client, operationError := hospitalA.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, operationError)

	_, operationError = client.SearchByID(context.Background(), "bad id!")
	require.ErrorIs(t, operationError, apperr.ErrInvalidInput)
	require.False(t, called)
}

func TestValidID(t *testing.T) {
	t.Parallel()
	require.True(t, hospitalA.ValidID("A123"))
	require.False(t, hospitalA.ValidID(""))
	require.False(t, hospitalA.ValidID("has space"))
	require.False(t, hospitalA.ValidID(string(make([]byte, 65))))
}
