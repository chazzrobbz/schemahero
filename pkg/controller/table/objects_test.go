package table

import (
	"testing"

	databasesv1alpha4 "github.com/schemahero/schemahero/pkg/apis/databases/v1alpha4"
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_planConfigMap(t *testing.T) {
	tests := []struct {
		name     string
		table    schemasv1alpha4.Table
		database databasesv1alpha4.Database
		expect   string
	}{
		{
			name: "basic test",
			table: schemasv1alpha4.Table{
				Spec: schemasv1alpha4.TableSpec{
					Database: "db",
					Name:     "name",
					Schema: &schemasv1alpha4.TableSchema{
						Postgres: &schemasv1alpha4.SQLTableSchema{},
					},
				},
			},
			database: databasesv1alpha4.Database{},
			expect: `database: db
name: name
schema:
  postgres: {}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := getPlanConfigMap(&test.database, &test.table)
			req.NoError(err)

			// check some of the fields on the config map
			assert.Equal(t, actual.Data["table.yaml"], test.expect)
		})
	}
}

func Test_vaultAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		expectedAnnotations map[string]string
		expectedArgs        []string
		database            *databasesv1alpha4.Database
		table               *schemasv1alpha4.Table
	}{
		{
			name: "Configures correctly when using Vault",
			expectedAnnotations: map[string]string{
				"vault.hashicorp.com/agent-inject":                      "true",
				"vault.hashicorp.com/agent-inject-secret-schemaherouri": "database/creds/schemahero",
				"vault.hashicorp.com/role":                              "schemahero-plan",
				"vault.hashicorp.com/agent-inject-template-schemaherouri": `
{{- with secret "database/creds/schemahero" -}}
postgres://{{ .Data.username }}:{{ .Data.password }}@postgres:5432/my-database{{- end }}`,
			},
			expectedArgs: []string{
				"plan",
				"--driver",
				"postgres",
				"--spec-file",
				"/specs/table.yaml",
				"--vault-uri-ref",
				"/vault/secrets/schemaherouri",
			},
			database: &databasesv1alpha4.Database{
				TypeMeta:   v1.TypeMeta{APIVersion: "databases.schemahero.io/v1alpha4", Kind: "Database"},
				ObjectMeta: v1.ObjectMeta{Name: "my-database"},
				Spec: databasesv1alpha4.DatabaseSpec{
					Connection: databasesv1alpha4.DatabaseConnection{
						Postgres: &databasesv1alpha4.PostgresConnection{
							URI: databasesv1alpha4.ValueOrValueFrom{
								ValueFrom: &databasesv1alpha4.ValueFrom{
									Vault: &databasesv1alpha4.Vault{
										Secret: "database/creds/schemahero",
										Role:   "schemahero-plan",
									},
								},
							},
						},
					},
				},
				Status: databasesv1alpha4.DatabaseStatus{},
			},
			table: &schemasv1alpha4.Table{
				TypeMeta:   v1.TypeMeta{APIVersion: "schemas.schemahero.io/v1alpha4", Kind: "Table"},
				ObjectMeta: v1.ObjectMeta{Name: "my-table"},
				Spec: schemasv1alpha4.TableSpec{
					Database: "my-database",
					Name:     "my-table",
					Schema: &schemasv1alpha4.TableSchema{
						Postgres: &schemasv1alpha4.SQLTableSchema{
							PrimaryKey: []string{"id"},
							Columns: []*schemasv1alpha4.SQLTableColumn{
								{
									Name: "id",
									Type: "text",
									Constraints: &schemasv1alpha4.SQLTableColumnConstraints{
										NotNull: new(bool),
									},
								},
								{
									Name: "name",
									Type: "text",
									Constraints: &schemasv1alpha4.SQLTableColumnConstraints{
										NotNull: new(bool),
									},
								},
							},
						},
					},
				},
				Status: schemasv1alpha4.TableStatus{},
			},
		},
		{
			name:                "Configures correctly when not using vault",
			expectedAnnotations: nil,
			expectedArgs: []string{
				"plan",
				"--driver",
				"postgres",
				"--spec-file",
				"/specs/table.yaml",
				"--uri",
				"postgres://user:password@postgres:5432/my-database",
			},
			database: &databasesv1alpha4.Database{
				TypeMeta:   v1.TypeMeta{APIVersion: "databases.schemahero.io/v1alpha4", Kind: "Database"},
				ObjectMeta: v1.ObjectMeta{Name: "my-database"},
				Spec: databasesv1alpha4.DatabaseSpec{
					Connection: databasesv1alpha4.DatabaseConnection{
						Postgres: &databasesv1alpha4.PostgresConnection{
							URI: databasesv1alpha4.ValueOrValueFrom{
								Value: "postgres://user:password@postgres:5432/my-database",
							},
						},
					},
				},
				Status: databasesv1alpha4.DatabaseStatus{},
			},
			table: &schemasv1alpha4.Table{
				TypeMeta:   v1.TypeMeta{APIVersion: "schemas.schemahero.io/v1alpha4", Kind: "Table"},
				ObjectMeta: v1.ObjectMeta{Name: "my-table"},
				Spec: schemasv1alpha4.TableSpec{
					Database: "my-database",
					Name:     "my-table",
					Schema: &schemasv1alpha4.TableSchema{
						Postgres: &schemasv1alpha4.SQLTableSchema{
							PrimaryKey: []string{"id"},
							Columns: []*schemasv1alpha4.SQLTableColumn{
								{
									Name: "id",
									Type: "text",
									Constraints: &schemasv1alpha4.SQLTableColumnConstraints{
										NotNull: new(bool),
									},
								},
								{
									Name: "name",
									Type: "text",
									Constraints: &schemasv1alpha4.SQLTableColumnConstraints{
										NotNull: new(bool),
									},
								},
							},
						},
					},
				},
				Status: schemasv1alpha4.TableStatus{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			r := &ReconcileTable{}
			actual, err := r.getPlanPod(test.database, test.table)
			assert.NoError(t, err)

			actualAnnotations := actual.ObjectMeta.Annotations
			actualArgs := actual.Spec.Containers[0].Args

			assert.Equal(t, test.expectedAnnotations, actualAnnotations)
			assert.Equal(t, test.expectedArgs, actualArgs)
		})
	}
}