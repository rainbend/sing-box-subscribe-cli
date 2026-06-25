package subconv

import "time"

type Options struct {
	URL             string
	Template        string
	Output          string
	Tag             string
	UserAgent       string
	Prefix          string
	ExcludeProtocol string
	ExcludeNodeName string
	OnlyNodes       bool
	Timeout         time.Duration
}

type Result struct {
	Config    any
	NodeCount int
	Warnings  []string
}

type outbound = map[string]any
