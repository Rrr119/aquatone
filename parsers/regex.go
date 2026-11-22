package parsers

import (
	"bufio"
	"io"

	"github.com/michenriksen/aquatone/core"
	"github.com/mvdan/xurls"
)

type RegexParser struct{}

func NewRegexParser() *RegexParser {
	return &RegexParser{}
}

func (p *RegexParser) Parse(r io.Reader) ([]string, error) {
	var targets []string

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		for _, target := range xurls.Relaxed().FindAllString(scanner.Text(), -1) {
			targets = append(targets, target)
		}
	}
	return core.DeduplicateTargets(targets), nil
}
