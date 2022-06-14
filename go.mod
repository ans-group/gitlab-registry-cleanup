module github.com/ukfast/gitlab-registry-cleanup

go 1.15

require (
	github.com/cheggaaa/pb v1.0.29
	github.com/mattn/go-runewidth v0.0.7 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.7.2
	github.com/xanzy/go-gitlab v0.39.0
	gopkg.in/yaml.v2 v2.4.0
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.4.0
