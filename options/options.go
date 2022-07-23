package options

import (
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Destination        string `short:"d" long:"destination" description:"Destination file path" required:"true"`
	UserCFSource       string `long:"cf" description:"User conversion functions source/package" required:"false"`
	UserCFPackageAlias string `long:"cf-alias" description:"User conversion functions package alias" required:"false"`
	FromName           string `long:"from" description:"Model from name" required:"true"`
	FromTag            string `long:"from-tag" description:"Model from tag" default:"map" required:"false"`
	FromSource         string `long:"from-source" description:"From model source/package" default:"." required:"false"`
	FromPackageAlias   string `long:"from-alias" description:"From model package alias" required:"false"`
	ToName             string `long:"to" description:"Model to name" required:"true"`
	ToTag              string `long:"to-tag" description:"Model to tag" default:"map" required:"false"`
	ToSource           string `long:"to-source" description:"To model source/package" default:"." required:"false"`
	ToPackageAlias     string `long:"to-alias" description:"To model package alias" required:"false"`
}

func ParseOptions() (Options, error) {
	var options Options
	_, err := flags.Parse(&options)
	return options, err
}
