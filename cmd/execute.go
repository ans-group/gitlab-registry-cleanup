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
	cmd.Flags().StringSlice("policy", []string{""}, "Limit policies to execute")

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

	log.Info("Retrieving all projects")
	projects, err := getAllProjects(client)
	if err != nil {
		return fmt.Errorf("Failed to retrieve projects: %s", err)
	}

	errors := false
	for _, repositoryConfig := range cfg.Repositories {
		err := processRepositoryConfig(cmd, client, projects, cfg, repositoryConfig)
		if err != nil {
			log.Error("Failed to process repository: %s", err)
			errors = true
		}
	}

	if errors {
		return fmt.Errorf("One or more errors occurred processing repositories")
	}

	return nil
}

func processRepositoryConfig(cmd *cobra.Command, client *gitlab.Client, projects []*gitlab.Project, cfg *config.Config, repositoryConfig config.RepositoryConfig) error {
	log.WithFields(log.Fields{
		"project": repositoryConfig.Project,
		"group":   repositoryConfig.Group,
	}).Info("Processing repository config")
	projectIDs, err := getRepositoryProjects(cmd, client, projects, repositoryConfig)
	if err != nil {
		return fmt.Errorf("Failed retrieving repository projects: %s", err)
	}

	err = processRepositoryProjects(cmd, client, cfg, repositoryConfig, projectIDs)
	if err != nil {
		return fmt.Errorf("Failed to process repository config projects: %s", err)
	}

	log.WithFields(log.Fields{
		"project": repositoryConfig.Project,
		"group":   repositoryConfig.Group,
	}).Info("Finished processing repository config")

	return nil
}

func getRepositoryProjects(cmd *cobra.Command, client *gitlab.Client, projects []*gitlab.Project, repositoryConfig config.RepositoryConfig) ([]int, error) {
	var projectIDs []int

	for _, project := range projects {
		// Skip if container registry not enabled
		if !project.ContainerRegistryEnabled {
			log.Tracef("Container registry not enabled for project %d", project.ID)
			continue
		}

		// If project, check if match
		if repositoryConfig.Project > 0 && repositoryConfig.Project != project.ID {
			continue
		}

		// If group, check if match
		if repositoryConfig.Group > 0 {
			groupIDs := []int{project.Namespace.ID}

			if repositoryConfig.Recurse {
				parentGroupIDs, err := getParentGroupIDsRecursive(client, repositoryConfig.Group)
				if err != nil {
					return nil, err
				}
				groupIDs = append(groupIDs, parentGroupIDs...)
			}

			if !intInSlice(repositoryConfig.Group, groupIDs) {
				continue
			}
		}

		projectIDs = append(projectIDs, project.ID)
	}

	return projectIDs, nil
}

func getParentGroupIDsRecursive(client *gitlab.Client, id int) ([]int, error) {
	var ids []int
	next := id
	for {
		namespace, _, err := client.Namespaces.GetNamespace(next)
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve namespace: %s", err)
		}
		if namespace.ParentID == 0 {
			break
		}

		ids = append(ids, namespace.ParentID)
		next = namespace.ParentID
	}

	return ids, nil
}

func processRepositoryProjects(cmd *cobra.Command, client *gitlab.Client, cfg *config.Config, repositoryConfig config.RepositoryConfig, projectIDs []int) error {
	log.Debugf("Processing %d repository projects", len(projectIDs))
	for _, projectID := range projectIDs {
		log.Debugf("Retrieving all Gitlab registry repositories for project %d", projectID)
		repositories, err := getAllProjectRepositories(client, projectID)
		if err != nil {
			return fmt.Errorf("Error retrieving all Gitlab registry repositories for project %d: %s", projectID, err)
		}

		log.Debugf("Found %d registry repositories", len(repositories))

		for _, repository := range repositories {
			if repositoryConfig.Images == nil || stringInSlice(repository.Path, repositoryConfig.Images) {
				log.Infof("Processing repository %s", repository.Path)
				err := processRepositoryProjectPolicies(cmd, client, cfg, repository, repositoryConfig, projectID)
				if err != nil {
					return err
				}
				log.Infof("Finished processing repository %s", repository.Path)
			} else {
				log.Debugf("Skipping unmatched repository %s", repository.Path)
			}
		}
	}

	log.Debug("Finished processing repository projects")

	return nil
}

func stringInSlice(str string, slice []string) bool {
	for _, sliceStr := range slice {
		if str == sliceStr {
			return true
		}
	}
	return false
}

func intInSlice(v int, slice []int) bool {
	for _, sliceInt := range slice {
		if v == sliceInt {
			return true
		}
	}
	return false
}

func processRepositoryProjectPolicies(cmd *cobra.Command, client *gitlab.Client, cfg *config.Config, repository *gitlab.RegistryRepository, repositoryConfig config.RepositoryConfig, projectID int) error {
	var policyFilter []string
	if cmd.Flags().Changed("policy") {
		policyFilter, _ = cmd.Flags().GetStringSlice("policy")
	}
	for _, policyName := range repositoryConfig.Policies {
		log.Infof("Processing repository policy %s", policyName)

		if len(policyFilter) > 0 && !stringInSlice(policyName, policyFilter) {
			log.Warnf("Skipping policy %s as not specified in policy flag", policyName)
			continue
		}

		policyCfg, err := cfg.GetPolicyConfig(policyName)
		if err != nil {
			return err
		}

		err = processRepositoryProjectPolicy(cmd, client, repository, projectID, policyCfg)
		if err != nil {
			return err
		}

		log.Infof("Finished processing repository policy %s", policyName)
	}

	return nil
}

func processRepositoryProjectPolicy(cmd *cobra.Command, client *gitlab.Client, repository *gitlab.RegistryRepository, projectID int, policyCfg config.PolicyConfig) error {
	log.Debug("Retrieving tag metadata")
	tagsMeta, err := getAllProjectRepositoryTags(client, repository, projectID)
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
		tag, _, err := client.ContainerRegistry.GetRegistryRepositoryTagDetail(projectID, repository.ID, tagMeta.Name)
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
				log.Info(logLine)
				_, err := client.ContainerRegistry.DeleteRegistryRepositoryTag(projectID, repository.ID, filteredTag.Name)
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

func getAllProjectRepositories(client *gitlab.Client, projectId int) ([]*gitlab.RegistryRepository, error) {
	var allRepositories []*gitlab.RegistryRepository
	page := 1
	for {
		log.WithField("page", page).Trace(("Retrieving repositories"))
		repositories, resp, err := client.ContainerRegistry.ListRegistryRepositories(projectId, &gitlab.ListRegistryRepositoriesOptions{Page: page})
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

func getAllProjectRepositoryTags(client *gitlab.Client, repository *gitlab.RegistryRepository, projectId int) ([]*gitlab.RegistryRepositoryTag, error) {
	var allTags []*gitlab.RegistryRepositoryTag
	page := 1
	for {
		log.WithField("page", page).Trace(("Retrieving tags"))
		tags, resp, err := client.ContainerRegistry.ListRegistryRepositoryTags(projectId, repository.ID, &gitlab.ListRegistryRepositoryTagsOptions{Page: page})
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

func getAllProjects(client *gitlab.Client) ([]*gitlab.Project, error) {
	var allProjects []*gitlab.Project
	page := 1
	for {
		log.WithField("page", page).Trace(("Retrieving projects"))
		projects, resp, err := client.Projects.ListProjects(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		})
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		page++
	}

	return allProjects, nil
}
