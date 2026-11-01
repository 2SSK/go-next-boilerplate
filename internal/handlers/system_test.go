package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/2SSK/go-next-boilerplate/api"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestGetReadyz(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
		want    any
	}{
		{"database up", nil, api.GetReadyz200JSONResponse{Status: api.ReadinessStatusOk, Database: api.ReadinessDatabaseUp}},
		{"database down", errors.New("connection refused"), api.GetReadyz503JSONResponse{Status: api.ReadinessStatusUnavailable, Database: api.ReadinessDatabaseDown}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSystemHandler("test", fakePinger{tt.pingErr}).GetReadyz(context.Background(), api.GetReadyzRequestObject{})
			if err != nil || got != tt.want {
				t.Errorf("GetReadyz = %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}
}

func TestGetLivezIgnoresTheDatabase(t *testing.T) {
	got, err := NewSystemHandler("test", fakePinger{errors.New("down")}).GetLivez(context.Background(), api.GetLivezRequestObject{})
	if err != nil || got != (api.GetLivez200JSONResponse{Status: api.LivenessStatusOk}) {
		t.Errorf("GetLivez = %#v, %v; want ok even with the database down", got, err)
	}
}
