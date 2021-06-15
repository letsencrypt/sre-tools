package main

import (
	"reflect"
	"testing"
)

func Test_validateConf(t *testing.T) {
	type args struct {
		conf           Conf
		configFilename string
	}
	tests := []struct {
		name    string
		args    args
		want    *Conf
		wantErr bool
	}{
		// Happy paths.
		{
			"valid, no ACL",
			args{Conf{"some_secret", "some_id", "some_region", "some_bucket", ""}, "some_file.yml"},
			&Conf{"some_secret", "some_id", "some_region", "some_bucket", ""},
			false,
		},
		{
			"valid, with ACL",
			args{Conf{"some_secret", "some_id", "some_region", "some_bucket", "bucket-owner-full-control"}, "some_file.yml"},
			&Conf{"some_secret", "some_id", "some_region", "some_bucket", "bucket-owner-full-control"},
			false,
		},
		// Sad paths.
		{
			"invalid, due to invalid ACL",
			args{Conf{"some_secret", "some_id", "some_region", "some_bucket", "bucket-owner-medium-control"}, "some_file.yml"},
			nil,
			true,
		},
		{
			"invalid, due to missing secret",
			args{Conf{"", "some_id", "some_region", "some_bucket", ""}, "some_file.yml"},
			nil,
			true,
		},
		{
			"invalid, due to missing id",
			args{Conf{"some_secret", "", "some_region", "some_bucket", ""}, "some_file.yml"},
			nil,
			true,
		},
		{
			"invalid, due to missing region",
			args{Conf{"some_secret", "some_id", "", "some_bucket", ""}, "some_file.yml"},
			nil,
			true,
		},
		{
			"invalid, due to missing bucket",
			args{Conf{"some_secret", "some_id", "some_region", "", ""}, "some_file.yml"},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateConf(tt.args.conf, tt.args.configFilename)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
