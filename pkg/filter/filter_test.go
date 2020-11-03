package filter

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/config"
	"github.com/xanzy/go-gitlab"
)

func TestNewFilterPipeline_ReturnsStruct(t *testing.T) {
	p := NewFilterPipeline(nil, config.FilterConfig{})

	assert.NotNil(t, p)
}

func TestFilterPipeline_Execute(t *testing.T) {
	t.Run("NoPipelineError_ReturnsNoError", func(t *testing.T) {
		f := func(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
			return tags, nil
		}

		p := NewFilterPipeline(nil, config.FilterConfig{})
		_, err := p.Execute(f)

		assert.Nil(t, err)
	})

	t.Run("PipelineError_ReturnsError", func(t *testing.T) {
		f := func(tags []*gitlab.RegistryRepositoryTag, config config.FilterConfig) ([]*gitlab.RegistryRepositoryTag, error) {
			return nil, errors.New("test error")
		}

		p := NewFilterPipeline(nil, config.FilterConfig{})
		_, err := p.Execute(f)

		assert.NotNil(t, err)
	})
}

func TestIncludeFilter(t *testing.T) {
	t.Run("NoIncludeSpecified_IncludesNone", func(t *testing.T) {
		result, err := IncludeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{})

		assert.Nil(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("IncludeSpecified_IncludesExpected", func(t *testing.T) {
		result, err := IncludeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Include: "test1.+",
		})

		assert.Nil(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, result[0].Name, "test12")
		assert.Equal(t, result[1].Name, "test123")
	})

	t.Run("RegexError_ReturnsError", func(t *testing.T) {
		_, err := IncludeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Include: "(",
		})

		assert.NotNil(t, err)
	})
}

func TestExcludeFilter(t *testing.T) {
	t.Run("NoExcludeSpecified_IncludesAll", func(t *testing.T) {
		result, err := ExcludeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{})

		assert.Nil(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("ExcludeSpecified_ExcludesExpected", func(t *testing.T) {
		result, err := ExcludeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Exclude: "test1.+",
		})

		assert.Nil(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, result[0].Name, "test1")
	})

	t.Run("RegexError_ReturnsError", func(t *testing.T) {
		_, err := ExcludeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Exclude: "(",
		})

		assert.NotNil(t, err)
	})
}

func TestKeepFilter(t *testing.T) {
	t.Run("NoKeepSpecified_IncludesAll", func(t *testing.T) {
		result, err := KeepFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{})

		assert.Nil(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("KeepSpecified_KeepsExpected", func(t *testing.T) {
		result, err := KeepFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Keep: 1,
		})

		assert.Nil(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, result[0].Name, "test1")
		assert.Equal(t, result[1].Name, "test12")
	})

	t.Run("KeepSpecifiedEqualToTags_KeepsAll", func(t *testing.T) {
		result, err := KeepFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Keep: 3,
		})

		assert.Nil(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("KeepSpecifiedMoreThanTags_KeepsAll", func(t *testing.T) {
		result, err := KeepFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{
			Keep: 7,
		})

		assert.Nil(t, err)
		assert.Len(t, result, 0)
	})
}

func TestOrderedFilter(t *testing.T) {
	t.Run("CreatedAtPresent_Orders", func(t *testing.T) {
		time1 := time.Now().Add(-time.Duration(5*24) * time.Hour)
		time12 := time.Now().Add(-time.Duration(4*24) * time.Hour)
		time123 := time.Now().Add(-time.Duration(3*24) * time.Hour)
		result, err := OrderedFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name:      "test1",
				CreatedAt: &time1,
			},
			{
				Name:      "test123",
				CreatedAt: &time123,
			},
			{
				Name:      "test12",
				CreatedAt: &time12,
			},
		}, config.FilterConfig{})

		assert.Nil(t, err)
		assert.Equal(t, result[0].Name, "test1")
		assert.Equal(t, result[1].Name, "test12")
		assert.Equal(t, result[2].Name, "test123")
	})
}

func TestExcludeLatestFilter(t *testing.T) {
	t.Run("LatestPresent_Excludes", func(t *testing.T) {
		result, err := ExcludeLatestFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "latest",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{})

		assert.Nil(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, result[0].Name, "test1")
		assert.Equal(t, result[1].Name, "test123")
	})

	t.Run("LatestNotPresent_NoAction", func(t *testing.T) {
		result, err := ExcludeLatestFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name: "test1",
			},
			{
				Name: "test12",
			},
			{
				Name: "test123",
			},
		}, config.FilterConfig{})

		assert.Nil(t, err)
		assert.Len(t, result, 3)
	})
}

func TestAgeFilter(t *testing.T) {
	t.Run("AgePresent_ExcludesExpected", func(t *testing.T) {
		time1 := time.Now().Add(-time.Duration(5*24) * time.Hour)
		time12 := time.Now().Add(-time.Duration(4*24) * time.Hour)
		time123 := time.Now().Add(-time.Duration(3*24) * time.Hour)
		result, err := AgeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name:      "test1",
				CreatedAt: &time1,
			},
			{
				Name:      "test12",
				CreatedAt: &time12,
			},
			{
				Name:      "test123",
				CreatedAt: &time123,
			},
		}, config.FilterConfig{
			Age: 4,
		})

		assert.Nil(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, result[0].Name, "test1")
		assert.Equal(t, result[1].Name, "test12")
	})

	t.Run("AgePresentAndEarlierThanAll_ExcludesAll", func(t *testing.T) {
		time1 := time.Now().Add(-time.Duration(5*24) * time.Hour)
		time12 := time.Now().Add(-time.Duration(4*24) * time.Hour)
		time123 := time.Now().Add(-time.Duration(3*24) * time.Hour)
		result, err := AgeFilter([]*gitlab.RegistryRepositoryTag{
			{
				Name:      "test1",
				CreatedAt: &time1,
			},
			{
				Name:      "test12",
				CreatedAt: &time12,
			},
			{
				Name:      "test123",
				CreatedAt: &time123,
			},
		}, config.FilterConfig{
			Age: 7,
		})

		assert.Nil(t, err)
		assert.Len(t, result, 0)
	})
}
