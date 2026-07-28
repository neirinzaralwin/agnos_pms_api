package hospitala_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/client/hospitala"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
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

	c, err := hospitala.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, err)

	p, err := c.SearchByID(context.Background(), "1100700123456")
	require.NoError(t, err)
	require.NotNil(t, p.NationalID)
	require.Equal(t, "1100700123456", *p.NationalID)
	require.Equal(t, "M", *p.Gender)
}

func TestSearchByID_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c, err := hospitala.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, err)

	_, err = c.SearchByID(context.Background(), "MISSING")
	require.ErrorIs(t, err, platform.ErrNotFound)
}

func TestSearchByID_Upstream500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c, err := hospitala.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, err)

	_, err = c.SearchByID(context.Background(), "X")
	require.ErrorIs(t, err, platform.ErrUpstream)
}

func TestSearchByID_BadJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{bad`))
	}))
	t.Cleanup(srv.Close)

	c, err := hospitala.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, err)

	_, err = c.SearchByID(context.Background(), "X")
	require.ErrorIs(t, err, platform.ErrUpstream)
}

func TestSearchByID_Timeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := hospitala.New(srv.URL, 50*time.Millisecond, nil)
	require.NoError(t, err)

	_, err = c.SearchByID(context.Background(), "X")
	require.ErrorIs(t, err, platform.ErrUpstream)
}

func TestSearchByID_InvalidID(t *testing.T) {
	t.Parallel()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := hospitala.New(srv.URL, 2*time.Second, nil)
	require.NoError(t, err)

	_, err = c.SearchByID(context.Background(), "bad id!")
	require.ErrorIs(t, err, platform.ErrInvalidInput)
	require.False(t, called)
}

func TestValidID(t *testing.T) {
	t.Parallel()
	require.True(t, hospitala.ValidID("A123"))
	require.False(t, hospitala.ValidID(""))
	require.False(t, hospitala.ValidID("has space"))
	require.False(t, hospitala.ValidID(string(make([]byte, 65))))
}
