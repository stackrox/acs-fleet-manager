package arrays

import (
	"testing"
	"unicode"

	. "github.com/onsi/gomega"
)

func Test_StringFindFirst(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary       []string
		val       string
		predicate func(x string) bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "FindFirst success",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Red" },
			},
			want: 2,
		},
		{
			name: "FindFirst Not Found",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Blue" },
			},
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(FindFirstString(tt.args.ary, tt.args.predicate)).To(Equal(tt.want))
		})
	}
}

func Test_FilterStringSlice(t *testing.T) {
	RegisterTestingT(t)
	res := FilterStringSlice([]string{"this", "is", "Red", "Hat"}, func(x string) bool { return x != "" && unicode.IsUpper([]rune(x)[0]) })
	Expect(res).To(HaveLen(2))
	Expect(res).To(Equal([]string{"Red", "Hat"}))
}

func Test_StringFirstNonEmpty(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary []string
	}
	tests := []struct {
		name    string
		args    args
		wants   string
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				ary: []string{"", "", "", "Red", "Hat"},
			},
			wants:   "Red",
			wantErr: false,
		},
		{
			name: "Not Found",
			args: args{
				ary: []string{"", "", "", "", ""},
			},
			wants:   "",
			wantErr: true,
		},
		{
			name: "No Values",
			args: args{
				ary: []string{},
			},
			wants:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FirstNonEmpty(tt.args.ary...)
			Expect(val).To(Equal(tt.wants))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_StringFirstNonEmptyOrDefault(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary          []string
		defaultValue string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Value found: Red",
			args: args{
				ary:          []string{"", "", "Red", "Hat"},
				defaultValue: "Red Hat",
			},
			want: "Red",
		},
		{
			name: "Value not found: default: 'Red Hat'",
			args: args{
				ary:          []string{"", "", "", ""},
				defaultValue: "Red Hat",
			},
			want: "Red Hat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(FirstNonEmptyOrDefault(tt.args.defaultValue, tt.args.ary...)).To(Equal(tt.want))
		})
	}
}

func Test_Contains(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary   []string
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Value found",
			args: args{
				ary:   []string{"yellow", "green", "blue"},
				value: "green",
			},
			want: true,
		},
		{
			name: "Value not found",
			args: args{
				ary:   []string{"yellow", "green", "blue"},
				value: "red",
			},
			want: false,
		},
		{
			name: "Value not found - Empty Array",
			args: args{
				ary:   []string{},
				value: "red",
			},
			want: false,
		},
		{
			name: "Value not found - Nil Array",
			args: args{
				ary:   nil,
				value: "red",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(Contains(tt.args.ary, tt.args.value)).To(Equal(tt.want))
		})
	}
}
