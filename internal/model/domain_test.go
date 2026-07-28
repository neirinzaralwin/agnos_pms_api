package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
)

func TestParseHospitalCode(t *testing.T) {
	t.Parallel()
	h, err := model.ParseHospitalCode("  Hospital-A ")
	require.NoError(t, err)
	require.Equal(t, "hospital-a", h.String())

	_, err = model.ParseHospitalCode("  ")
	require.ErrorIs(t, err, model.ErrInvalidInput)
}

func TestParseGender(t *testing.T) {
	t.Parallel()
	m := "M"
	x := "X"
	require.Equal(t, "M", model.ParseGender(&m).String())
	require.False(t, model.ParseGender(&x).IsSet())
	require.False(t, model.ParseGender(nil).IsSet())
}

func TestParsePassword(t *testing.T) {
	t.Parallel()
	_, err := model.ParsePassword("short")
	require.ErrorIs(t, err, model.ErrInvalidInput)
	p, err := model.ParsePassword("password12345")
	require.NoError(t, err)
	require.Equal(t, "password12345", p.String())
}

func TestParseLookupID(t *testing.T) {
	t.Parallel()
	id, err := model.ParseLookupID("A123")
	require.NoError(t, err)
	require.Equal(t, "A123", id.String())
	_, err = model.ParseLookupID("bad id!")
	require.ErrorIs(t, err, model.ErrInvalidInput)
}

func TestRegisterFromHIS(t *testing.T) {
	t.Parallel()
	hospital, err := model.ParseHospitalCode("hospital-a")
	require.NoError(t, err)

	nid := "1100700123456"
	gender := "X"
	name := "Somchai"
	p, err := model.RegisterFromHIS(hospital, model.HISPatientData{
		FirstNameEN: &name,
		NationalID:  &nid,
		Gender:      &gender,
	})
	require.NoError(t, err)
	require.True(t, p.BelongsTo(hospital))
	require.Nil(t, p.Gender) // odd gender coerced
	require.True(t, p.HasIdentity())
}

func TestParseSearchCriteria_RequiresFilter(t *testing.T) {
	t.Parallel()
	_, err := model.ParseSearchCriteria(model.SearchCriteriaInput{})
	require.ErrorIs(t, err, model.ErrInvalidInput)

	email := "a@b.com"
	c, err := model.ParseSearchCriteria(model.SearchCriteriaInput{Email: &email})
	require.NoError(t, err)
	require.Equal(t, "a@b.com", *c.Email)
}

func TestNewStaff(t *testing.T) {
	t.Parallel()
	u, err := model.ParseUsername("alice")
	require.NoError(t, err)
	h, err := model.ParseHospitalCode("hospital-a")
	require.NoError(t, err)
	s, err := model.NewStaff(u, h, "hash")
	require.NoError(t, err)
	require.True(t, s.BelongsTo(h))
}
