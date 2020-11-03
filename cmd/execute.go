package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/config"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/filter"
	"github.com/ukfast/gitlab-registry-cleanup/pkg/progress"
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

	cmd.Flags().Bool("dry-run", false, "Specifies command should be ran in dry-run mode")
	cmd.Flags().Bool("progress", false, "Outputs progress")

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

	return processRepositories(cmd, client, cfg)
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

func processRepositories(cmd *cobra.Command, client *gitlab.Client, cfg *config.Config) error {
	for _, repositoryCfg := range cfg.Repositories {
		log.WithFields(log.Fields{
			"project_id": repositoryCfg.Project,
			"image":      repositoryCfg.Image,
		}).Infof("Processing repository %s", repositoryCfg.Image)

		repositories, err := getAllRepositories(client, repositoryCfg)
		if err != nil {
			return fmt.Errorf("Error retrieving all Gitlab registry repositories for project %d: %s", repositoryCfg.Project, err)
		}

		for _, repository := range repositories {
			if repositoryCfg.Image == repository.Path {
				err := processRepository(cmd, client, cfg, repository, repositoryCfg)
				if err != nil {
					return err
				}
			}
		}
		log.Infof("Finished processing repository %s", repositoryCfg.Image)
	}

	return nil
}

func processRepository(cmd *cobra.Command, client *gitlab.Client, cfg *config.Config, repository *gitlab.RegistryRepository, repositoryCfg config.RepositoryConfig) error {
	for _, policyName := range repositoryCfg.Policies {
		log.Infof("Processing repository policy %s", policyName)
		policyCfg, err := cfg.GetPolicyConfig(policyName)
		if err != nil {
			return err
		}

		err = processRepositoryPolicy(cmd, client, repository, repositoryCfg, policyCfg)
		if err != nil {
			return err
		}

		log.Infof("Finished processing repository policy %s", policyName)
	}

	return nil
}

func processRepositoryPolicy(cmd *cobra.Command, client *gitlab.Client, repository *gitlab.RegistryRepository, repositoryCfg config.RepositoryConfig, policyCfg config.PolicyConfig) error {
	log.Debug("Retrieving tag metadata")
	tagsMeta, err := getAllTags(client, repository, repositoryCfg)
	if err != nil {
		return fmt.Errorf("Failed retrieving tags: %w", err)
	}

	var tags []*gitlab.RegistryRepositoryTag

	log.Info("Retrieving tag details")

	progressFlag, _ := cmd.Flags().GetBool("progress")

	bar := progress.NewProgress(progressFlag, len(tagsMeta))
	bar.Start()
	for _, tagMeta := range tagsMeta {
		bar.Increment()
		log.Debugf("Retrieving details for tag %s", tagMeta.Name)
		tag, _, err := client.ContainerRegistry.GetRegistryRepositoryTagDetail(repositoryCfg.Project, repository.ID, tagMeta.Name)
		if err != nil {
			return fmt.Errorf("Failed retrieving tag detail: %w", err)
		}
		tags = append(tags, tag)
	}
	bar.Finish()

	log.WithFields(log.Fields{
		"include": policyCfg.Filter.Include,
		"exclude": policyCfg.Filter.Exclude,
		"keep":    policyCfg.Filter.Keep,
		"age":     policyCfg.Filter.Age,
	}).Debug("Executing filter pipeline")

	f := filter.NewFilterPipeline(tags, policyCfg.Filter)
	filteredTags, err := f.Execute(
		filter.ExcludeLatestFilter,
		filter.IncludeFilter,
		filter.OrderedFilter,
		filter.KeepFilter,
		filter.AgeFilter,
		filter.ExcludeFilter,
	)
	if err != nil {
		return fmt.Errorf("Failed to execute filter pipeline: %w", err)
	}

	log.Infof("Found %d tags for removal", len(filteredTags))
	if len(filteredTags) > 0 {
		log.Info("Removing tags")

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		bar = progress.NewProgress(progressFlag, len(filteredTags))
		bar.Start()
		for _, filteredTag := range filteredTags {
			bar.Increment()
			logLine := fmt.Sprintf("Removing tag %s", filteredTag.Name)
			if dryRun {
				log.Warnf("[DRY RUN]: %s", logLine)
			} else {
				log.Debug(logLine)
				_, err := client.ContainerRegistry.DeleteRegistryRepositoryTag(repositoryCfg.Project, repository.ID, filteredTag.Name)
				if err != nil {
					return fmt.Errorf("Failed to remove tag %s: %w", filteredTag.Name, err)
				}
			}
		}
		bar.Finish()

		log.Infof("Finished removing %d tags", len(filteredTags))
	}

	return nil
}
