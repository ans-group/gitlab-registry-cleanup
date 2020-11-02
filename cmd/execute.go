package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/config"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/filter"
	"github.com/xanzy/go-gitlab"
)

func ExecuteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Executes cleanup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeCleanup(cmd, args)
		},
	}

	cmd.Flags().Bool("dry-run", false, "specifies command should be ran in dry-run mode")

	return cmd
}

func executeCleanup(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}
	err := viper.Unmarshal(cfg)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal config: %w", err)
	}

	client, err := gitlab.NewClient(viper.GetString("access_token"), gitlab.WithBaseURL(viper.GetString("url")))
	if err != nil {
		return fmt.Errorf("Failed initialising Gitlab client: %s", err)
	}

	for _, repositoryCfg := range cfg.Repositories {
		log.Infof("Processing repository %s", repositoryCfg.Image)
		repositories, err := getAllRepositories(client, repositoryCfg)
		if err != nil {
			return fmt.Errorf("Error retrieving all Gitlab registry repositories for project %d: %s", repositoryCfg.Project, err)
		}

		for _, repository := range repositories {
			if repositoryCfg.Image == repository.Path {
				err := processRepository(cmd, client, repository, repositoryCfg)
				if err != nil {
					return fmt.Errorf("Failed processing repository %s: %s", repositoryCfg.Image, err)
				}
			}
		}
		log.Infof("Finished processing repository %s", repositoryCfg.Image)
	}

	return nil
}

func getAllRepositories(client *gitlab.Client, repositoryCfg config.RepositoryConfig) ([]*gitlab.RegistryRepository, error) {
	var allRepositories []*gitlab.RegistryRepository
	page := 1
	for {
		repositories, resp, err := client.ContainerRegistry.ListRegistryRepositories(repositoryCfg.Project, &gitlab.ListRegistryRepositoriesOptions{Page: page})
		if err != nil {
			return nil, err
		}

		allRepositories = append(allRepositories, repositories...)
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		page++
	}

	return allRepositories, nil
}

func getAllTags(client *gitlab.Client, repository *gitlab.RegistryRepository, repositoryCfg config.RepositoryConfig) ([]*gitlab.RegistryRepositoryTag, error) {
	var allTags []*gitlab.RegistryRepositoryTag
	page := 1
	for {
		tags, resp, err := client.ContainerRegistry.ListRegistryRepositoryTags(repositoryCfg.Project, repository.ID, &gitlab.ListRegistryRepositoryTagsOptions{Page: page})
		if err != nil {
			return nil, err
		}

		allTags = append(allTags, tags...)
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		page++
	}

	return allTags, nil
}

func processRepository(cmd *cobra.Command, client *gitlab.Client, repository *gitlab.RegistryRepository, repositoryCfg config.RepositoryConfig) error {
	tags, err := getAllTags(client, repository, repositoryCfg)
	if err != nil {
		return fmt.Errorf("Failed retrieving tags: %w", err)
	}

	f := filter.NewFilterPipeline(tags, repositoryCfg.Filter)

	filteredTags, err := f.Execute(
		filter.ExcludeLatestFilter,
		filter.IncludeFilter,
		filter.OrderedFilter,
		filter.KeepFilter,
		filter.AgeFilter,
		filter.ExcludeFilter,
	)

	if err != nil {
		return fmt.Errorf("Failed to execute filter: %w", err)
	}

	log.Infof("Found %d tags to remove", len(filteredTags))
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, filteredTag := range filteredTags {
		logLine := fmt.Sprintf("Removing tag %s", filteredTag.Name)
		if dryRun {
			log.Infof("[DRY RUN]: %s", logLine)
		} else {
			log.Info(logLine)
			_, err := client.ContainerRegistry.DeleteRegistryRepositoryTag(repositoryCfg.Project, repository.ID, filteredTag.Name)
			if err != nil {
				return fmt.Errorf("Failed to remove tag %s: %w", filteredTag.Name, err)
			}
		}
	}

	log.Infof("Finished removing %d tags", len(filteredTags))

	return nil
}
