package services

import (
	"fmt"
	"testing"

	gomocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

// dbMock is a function that sets up db mocks and return a function to validate if the mocks were called properly
type dbMock func() (validate func() error)

func TestSetDefaultVersion(t *testing.T) {
	connectionFactory := db.NewMockConnectionFactory(nil)

	tests := []struct {
		name         string
		versionInput string
		setupdbMock  dbMock
		wantErr      bool
	}{
		{
			name:         "returns select errors",
			versionInput: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			setupdbMock: func() (validate func() error) {
				gomocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithError(fmt.Errorf("database error"))
				return nil
			},
			wantErr: true,
		},
		{
			name:         "return no error and does not call create for version string equals returned version string",
			versionInput: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			setupdbMock: func() (validate func() error) {
				res := []map[string]interface{}{
					{
						"id":      2,
						"version": "quay.io/rhacs-eng/stackrox-operator:3.73.3",
					},
				}
				gomocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithReply(res)
				createMock := gomocket.Catcher.NewMock().WithQuery(`INSERT INTO "central_default_versions`)
				return func() error {
					if createMock.Triggered == true {
						return fmt.Errorf("expected Create to not be triggered, but it was")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:         "returns Create errors",
			versionInput: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			setupdbMock: func() (validate func() error) {
				res := []map[string]interface{}{
					{
						"id":      2,
						"version": "quay.io/rhacs-eng/stackrox-operator:3.73.2",
					},
				}
				gomocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithReply(res)
				gomocket.Catcher.NewMock().WithQuery(`INSERT INTO "central_default_versions"`).WithError(fmt.Errorf("database error"))
				return nil
			},
			wantErr: true,
		},
		{
			name:         "calls Create for new version string",
			versionInput: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			setupdbMock: func() (validate func() error) {
				res := []map[string]interface{}{
					{
						"id":      2,
						"version": "quay.io/rhacs-eng/stackrox-operator:3.73.2",
					},
				}
				gomocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithReply(res)
				createMock := gomocket.Catcher.NewMock().WithQuery(`INSERT INTO "central_default_versions"`)
				return func() error {
					if createMock.Triggered == false {
						return fmt.Errorf("expected Create to be called, but was not")
					}
					return nil
				}
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var validateDbQueries func() error
			if tc.setupdbMock != nil {
				validateDbQueries = tc.setupdbMock()
			}

			versionService := NewCentralDefaultVersionService(connectionFactory, nil)

			err := versionService.SetDefaultVersion(tc.versionInput)

			if (err != nil) != tc.wantErr {
				t.Fatalf("SetDefaultVersion error = %v, wantErr = %v", err, tc.wantErr)
				return
			}

			if validateDbQueries != nil {
				err := validateDbQueries()
				if err != nil {
					t.Fatalf("error validating db queries: %s", err)
				}
			}
		})
	}
}

func TestGetDefaultVersion(t *testing.T) {
	connectionFactory := db.NewMockConnectionFactory(nil)

	tests := []struct {
		name            string
		expectedVersion string
		setupdbMock     dbMock
		wantErr         bool
	}{
		{
			name:            "returns select errors",
			expectedVersion: "",
			setupdbMock: func() (validate func() error) {
				gomocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithError(fmt.Errorf("database error"))
				return nil
			},
			wantErr: true,
		},
		{
			name:            "returns version returned by db",
			expectedVersion: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			setupdbMock: func() (validate func() error) {
				res := []map[string]interface{}{
					{
						"id":      2,
						"version": "quay.io/rhacs-eng/stackrox-operator:3.73.3",
					},
				}
				gomocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithReply(res)
				return nil
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var validateDbQueries func() error
			if tc.setupdbMock != nil {
				validateDbQueries = tc.setupdbMock()
			}

			versionService := NewCentralDefaultVersionService(connectionFactory, nil)

			version, err := versionService.GetDefaultVersion()

			if (err != nil) != tc.wantErr {
				t.Fatalf("GetDefaultVersion error = %v, wantErr = %v", err, tc.wantErr)
				return
			}

			if tc.expectedVersion != version {
				t.Fatalf("expected GetDefaultVersion to return %s, got %s", tc.expectedVersion, version)
			}

			if validateDbQueries != nil {
				err := validateDbQueries()
				if err != nil {
					t.Fatalf("error validating db queries: %s", err)
				}
			}
		})
	}
}

func TestServiceStart(t *testing.T) {
	connectionFactory := db.NewMockConnectionFactory(nil)

	tests := []struct {
		name        string
		inputConfig config.CentralConfig
		setupdbMock dbMock
		wantExit    bool
	}{
		{
			name: "exit for validation error",
			inputConfig: config.CentralConfig{
				CentralDefaultVersion: "wrong.registry/rhacs-eng/stackrox-operator:3.73.3",
			},
			wantExit: true,
		},
		{
			name: "exit for db errors",
			inputConfig: config.CentralConfig{
				CentralDefaultVersion: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			},
			setupdbMock: func() (validate func() error) {
				gomocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithError(fmt.Errorf("database error"))
				return nil
			},
			wantExit: true,
		},
		{
			name: "calls Create for valid version",
			inputConfig: config.CentralConfig{
				CentralDefaultVersion: "quay.io/rhacs-eng/stackrox-operator:3.73.3",
			},
			setupdbMock: func() (validate func() error) {
				res := []map[string]interface{}{
					{
						"id":      2,
						"version": "quay.io/rhacs-eng/stackrox-operator:3.73.2",
					},
				}
				gomocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(res)
				createMock := gomocket.Catcher.NewMock().WithQuery("INSERT INTO")
				return func() error {
					if !createMock.Triggered {
						return fmt.Errorf("expected Create to be triggered, but was not")
					}
					return nil
				}
			},
			wantExit: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exitCalled := false
			oldOsExit := osExit
			osExit = func(code int) {
				exitCalled = true
			}
			defer func() {
				osExit = oldOsExit
			}()

			var validateDbQueries func() error
			if tc.setupdbMock != nil {
				validateDbQueries = tc.setupdbMock()
			}

			versionService := NewCentralDefaultVersionService(connectionFactory, &tc.inputConfig)
			versionService.Start()

			if tc.wantExit != exitCalled {
				t.Fatalf("osExit called: %v, wanted: %v", exitCalled, tc.wantExit)
			}

			if validateDbQueries != nil {
				err := validateDbQueries()
				if err != nil {
					t.Fatalf("error validating db queries: %s", err)
				}
			}
		})
	}

}
