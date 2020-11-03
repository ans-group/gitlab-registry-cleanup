# gitlab-registry-cleanup

A small utility for cleaning up Gitlab container registry image tags, inspired by the built in registry cleanup policies built into Gitlab.


## Why?

When attempting to implement [Cleanup Policies](https://docs.gitlab.com/ee/user/packages/container_registry/#cleanup-policy), we found configuration options to be limited, with only one policy per repository currently possible. This application implements the same functionality, however allows many policies to be specified.
Another drawback to the builtin cleanup policies is the lack of methods for validating policies. This application addresses this problem with the ability of executing policies in dry-run mode.

## Usage

```
A tool for cleaning up gitlab registries

Usage:
  gitlab-registry-cleanup [command]

Available Commands:
  execute     Executes cleanup
  help        Help about any command

Flags:
      --config string   config file (default "config.yml")
      --debug           specifies logging level should be set to debug
  -h, --help            help for gitlab-registry-cleanup
```

## Commands

**execute**

Executes cleanup process

#### Flags

* `--dry-run`: Specifies execution should be ran in dry run mode. Tag deletions will not occur


## Config

Application config is specified with a yaml configuration file, with an example below:

```yaml
access_token: myaccesstoken
url: https://gitlab.privateinstance.com
policies:
- name: nonsemverpolicy
  filter:
    include: .*
    exclude: ^v.+
    keep: 5
    age: 30
repositories:
- project: 123
  image: myproject/somerepository
  policies:
  - nonsemverpolicy
```

* `access_token`: Private access token with `api` read/write scope
* `url`: Gitlab instance URL
* `policies`: __array__
  * `name`: Name of policy
  * `filter`: __object__
    * `include`: Regex specifying image tags to include - no tags will be matched if this isn't specified
    * `exclude`: (Optional) Regex specifying image tags to exclude
    * `keep`: (Optional) Specifies amount of tags to keep
    * `age`: (Optional) Specifies amount of days to keep tags
* `repositories` __array__
  * `project`: Project ID to target
  * `image`: Path of repository/image
  * `policies` __array__
    * Name of policies

## Docker

We recommend using Docker for executing this utility. Example usage can be found below:

```
docker run -v "${PWD}/config.yml:/config.yml" --rm -it ukfast/gitlab-registry-cleanup execute --config /config.yml
```
