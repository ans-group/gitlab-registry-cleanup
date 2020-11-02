package filter

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/config"
	"github.com/xanzy/go-gitlab"
)

type Filter func(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error)

type FilterPipeline struct {
	tags   []*gitlab.RegistryRepositoryTag
	config config.FilterConfig
}

func NewFilterPipeline(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) *FilterPipeline {
	return &FilterPipeline{
		tags:   tags,
		config: config,
	}
}

func (f *FilterPipeline) Execute(filters ...Filter) ([]*gitlab.RegistryRepositoryTag, error) {
	filteredTags := f.tags
	for _, filter := range filters {
		filteredTagsResult, err := filter(filteredTags, f.config)
		if err != nil {
			return filteredTagsResult, err
		}
		filteredTags = filteredTagsResult
	}

	return filteredTags, nil
}

func IncludeFilter(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	var filteredTags []*gitlab.RegistryRepositoryTag

	for _, tag := range tags {
		included := true
		if len(config.Include) > 0 {
			matched, err := regexp.MatchString(config.Include, tag.Name)
			if err != nil {
				return filteredTags, fmt.Errorf("Regexp match failed with include %s for tag %s: %s", config.Include, tag.Name, err)
			}

			included = matched
		}

		if included {
			log.Debugf("IncludeFilter: Including matched tag %s", tag.Name)
			filteredTags = append(filteredTags, tag)
		}
	}

	return filteredTags, nil
}

func ExcludeFilter(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	var filteredTags []*gitlab.RegistryRepositoryTag

	for _, tag := range tags {
		excluded := false
		if len(config.Exclude) > 0 {
			matched, err := regexp.MatchString(config.Exclude, tag.Name)
			if err != nil {
				return filteredTags, fmt.Errorf("Regexp match failed with exclude %s for tag %s: %s", config.Exclude, tag.Name, err)
			}

			excluded = matched
		}

		if !excluded {
			log.Debugf("ExcludeFilter: Including non-excluded tag %s", tag.Name)
			filteredTags = append(filteredTags, tag)
		}
	}

	return filteredTags, nil
}

func KeepFilter(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	var filteredTags []*gitlab.RegistryRepositoryTag

	var notKept []*gitlab.RegistryRepositoryTag
	if config.Keep > 0 {
		if config.Keep <= len(tags) {
			notKept = tags[:len(tags)-config.Keep]
		}
	} else {
		notKept = tags[:]
	}

	for _, tag := range notKept {
		log.Debugf("KeepFilter: Including non-kept tag %s", tag.Name)
		filteredTags = append(filteredTags, tag)
	}

	return filteredTags, nil
}

func OrderedFilter(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	filteredTags := tags
	var err error

	sort.SliceStable(filteredTags, func(i, j int) bool {
		if filteredTags[i].CreatedAt == nil || filteredTags[j].CreatedAt == nil {
			return false
		}

		return filteredTags[i].CreatedAt.Before(*filteredTags[j].CreatedAt)
	})

	log.Debug("OrderedFilter: Ordering tags")
	return filteredTags, err
}

func ExcludeLatestFilter(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	var filteredTags []*gitlab.RegistryRepositoryTag

	for _, tag := range tags {
		if tag.Name != "latest" {
			log.Debugf("ExcludeLatestFilter: Including non-latest tag %s", tag.Name)
			filteredTags = append(filteredTags, tag)
		}
	}

	return filteredTags, nil
}

func AgeFilter(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	var filteredTags []*gitlab.RegistryRepositoryTag

	for _, tag := range tags {
		if config.Age < 1 || tag.CreatedAt.Before(time.Now().Add(-(time.Duration(config.Age*24) * time.Hour))) {
			log.Debugf("ExcludeLatestFilter: Including aged tag %s", tag.Name)
			filteredTags = append(filteredTags, tag)
		}
	}

	return filteredTags, nil
}
