package main

import (
	"reflect"
	"testing"
	"time"
)

func Test_loadConfig(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "defaults-all",
			args: args{file: "test-resources/config-test/defaults-all.yml"},
			want: &Config{Defaults: DefaultsData{
				DataSourceRef:     "my-ds",
				QueryInterval:     time.Minute * 15,
				QueryTimeout:      time.Minute * 5,
				QueryValueOnError: "-1",
			},
			},
			wantErr: false,
		},
		{
			name:    "empty-file",
			args:    args{file: "test-resources/config-test/empty.yml"},
			want:    newConfig(),
			wantErr: false,
		},
		{
			name:    "missing-driver-in-datasource",
			args:    args{file: "test-resources/config-test/datasource-missing-driver.yml"},
			wantErr: true,
		},
		{
			name:    "missing-properties-in-datasource",
			args:    args{file: "test-resources/config-test/datasource-missing-properties.yml"},
			wantErr: true,
		},
		{
			name:    "no-datasource",
			args:    args{file: "test-resources/config-test/datasource-no.yml"},
			want:    newConfig(),
			wantErr: false,
		},
		{
			name: "one-datasource",
			args: args{file: "test-resources/config-test/datasource-one.yml"},
			want: &Config{Defaults: createDefaultsData(),
				DataSources: map[string]DataSource{
					"mysql-test": DataSource{
						Driver: "mysql",
						Properties: map[string]interface{}{
							"host":     "localhost",
							"port":     3306,
							"user":     "root",
							"password": "unsecure",
							"database": "test",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "two-datasource",
			args: args{file: "test-resources/config-test/datasource-two.yml"},
			want: &Config{Defaults: createDefaultsData(),
				DataSources: map[string]DataSource{
					"mysql-test": DataSource{
						Driver: "mysql",
						Properties: map[string]interface{}{
							"host":     "localhost",
							"port":     3306,
							"user":     "root",
							"password": "unsecure",
							"database": "test",
						},
					},
					"mysql-test-2": DataSource{
						Driver: "mysql-2",
						Properties: map[string]interface{}{
							"host":     "localhost-2",
							"port":     3307,
							"user":     "root-2",
							"password": "unsecure-2",
							"database": "test-2",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadConfig(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//spew.Dump(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadConfig() = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_loadQueryConfig(t *testing.T) {
	type args struct {
		queriesFile string
		config      *Config
	}
	c, err := loadConfig("test-resources/config-test/queries-config.yml")
	if err != nil {
		t.Errorf("Failed to load config file: %s", err)
		return
	}
	tests := []struct {
		name    string
		args    args
		want    QueryList
		wantErr bool
	}{
		{
			name: "queries-datasource",
			args: args{
				queriesFile: "test-resources/config-test/queries-datasource.yml",
				config:      c,
			},
			want: []*Query{
				&Query{
					Name:          "query_ds_1",
					DataSourceRef: "my-ds-1",
					Driver:        "mysql",
					Connection: map[string]interface{}{
						"host":     "localhost",
						"port":     3306,
						"user":     "root",
						"password": "unsecure",
						"database": "test",
					},
					SQL:          "select 1 from dual\n",
					Params:       nil,
					Interval:     time.Second * 10,
					Timeout:      time.Second * 5,
					DataField:    "",
					ValueOnError: "0",
				},
				&Query{
					Name:          "query_ds_2",
					DataSourceRef: "my-ds-2",
					Driver:        "postgresql",
					Connection: map[string]interface{}{
						"host":     "localhost",
						"port":     5432,
						"user":     "postgres",
						"password": "unsecure",
						"database": "test",
					},
					SQL:          "select 1 from dual\n",
					Params:       nil,
					Interval:     time.Minute * 15,
					Timeout:      time.Minute * 5,
					DataField:    "",
					ValueOnError: "-1",
				},
			},
			wantErr: false,
		},
		{
			name: "queries-submetrics",
			args: args{
				queriesFile: "test-resources/config-test/queries-submetrics.yml",
				config:      c,
			},
			want: []*Query{
				&Query{
					Name:          "query_ds_1",
					SQL:           `select 1 as "sum", 2 as "count" from dual`,
					DataSourceRef: "my-ds-1",
					Driver:        "mysql",
					Connection: map[string]interface{}{
						"host":     "localhost",
						"port":     3306,
						"user":     "root",
						"password": "unsecure",
						"database": "test",
					},
					Interval: time.Minute * 15,
					Timeout:  time.Minute * 5,
					Params:   nil,
					SubMetrics: map[string]string{
						"count": "count",
						"sum":   "sum",
					},
					ValueOnError: "-1",
					DataField:    "",
				},
			},
			wantErr: false,
		},
		{
			name: "queries-compatibility",
			args: args{
				queriesFile: "test-resources/config-test/queries-compatibility.yml",
				config:      newConfig(),
			},
			want: []*Query{
				&Query{
					Name:   "num_products",
					Driver: "postgresql",
					Connection: map[string]interface{}{
						"host":     "example.org",
						"port":     5432,
						"user":     "postgres",
						"password": "s3cre7",
						"database": "products",
					},
					SQL:          "select 1 from dual\n",
					Params:       nil,
					Interval:     DefaultInterval,
					Timeout:      DefaultTimeout,
					DataField:    "",
					ValueOnError: "",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadQueryConfig(tt.args.queriesFile, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadQueryConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// spew.Dump(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadQueryConfig() = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_loadNonExistingFiles(t *testing.T) {
	file := "does-not-exist"
	_, err := loadConfig(file)
	if err == nil {
		t.Errorf("No errors even if config file [%s] does not exist!", file)
		return
	}

	_, err = loadQueryConfig(file, newConfig())
	if err == nil {
		t.Errorf("No errors even if query file [%s] does not exist!", file)
		return
	}

	_, err = loadQueriesInDir(file, newConfig(), false)
	if err == nil {
		t.Errorf("No errors even if query directory [%s] does not exist!", file)
		return
	}
}

func Test_allowBrokenQueryFileInDir(t *testing.T) {
	//config should allow loading queries from a directory that contains invalid files; bad files should not
	//stop good queries from running

	//load a directory having one good and one bad query
	q, err := loadQueriesInDir("test-resources/config-test/one-good-query", newConfig(), true)
	//expect no error and 1 query
	if err != nil {
		t.Fatal(err)
	}
	if len(q) != 1 {
		t.Fatal(len(q))
	}
}
