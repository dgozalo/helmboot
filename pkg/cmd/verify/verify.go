package verify

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jenkins-x-labs/helmboot/pkg/common"
	"github.com/jenkins-x-labs/helmboot/pkg/envfactory"
	"github.com/jenkins-x-labs/helmboot/pkg/githelpers"
	"github.com/jenkins-x-labs/helmboot/pkg/reqhelpers"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	verifyLong = templates.LongDesc(`
		Verifies the given environment git repository is setup correctly.

		Typicaly this is mostly used with Multi Cluster to verify the Staging / Production environment git repository is setup correctly
`)

	verifyExample = templates.Examples(`
		# verifies the staging repository is setup correctly
		%s verify --git-url=https://github.com/myorg/environment-mycluster-staging.git
	`)
)

// VerifyOptions the options for verifying
type VerifyOptions struct {
	envfactory.EnvFactory
	DisableVerifyPackages bool
	Flags                 reqhelpers.RequirementBools
	OverrideRequirements  config.RequirementsConfig
	Cmd                   *cobra.Command
	Args                  []string
	GitCloneURL           string
	Dir                   string
}

// NewCmdVerify creates a command object for the command
func NewCmdVerify() (*cobra.Command, *VerifyOptions) {
	o := &VerifyOptions{}

	cmd := &cobra.Command{
		Use:     "verify",
		Short:   "Verifies the given environment git repository is setup correctly",
		Long:    verifyLong,
		Example: fmt.Sprintf(verifyExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Cmd = cmd
	o.AddVerifyOptions(cmd)
	return cmd, o
}

func (o *VerifyOptions) AddVerifyOptions(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "", "", "The directory used to clone the git repository. If no directory is specified a temporary directory will be used")
	cmd.Flags().StringVarP(&o.GitCloneURL, "git-url", "", "", "The git repository to clone to upgrade")

	reqhelpers.AddRequirementsOptions(cmd, &o.OverrideRequirements)
	reqhelpers.AddRequirementsFlagsOptions(cmd, &o.Flags)
}

// Run implements the command
func (o *VerifyOptions) Run() error {
	if o.Gitter == nil {
		o.Gitter = gits.NewGitCLI()
	}

	dir, err := o.gitCloneIfRequired(o.Gitter)
	if err != nil {
		return err
	}

	err = reqhelpers.OverrideRequirements(o.Cmd, o.Args, dir, &o.OverrideRequirements, &o.Flags)
	if err != nil {
		return errors.Wrapf(err, "failed to override requirements in dir %s", dir)
	}

	err = o.EnvFactory.VerifyPreInstall(o.DisableVerifyPackages, dir)
	if err != nil {
		return errors.Wrapf(err, "failed to verify requirements in dir %s", dir)
	}

	changes, err := githelpers.AddAndCommitFiles(o.Gitter, dir, "fix: initial code")
	if err != nil {
		return err
	}
	if !changes {
		return nil
	}

	remote := "origin"
	err = o.Gitter.Push(dir, remote, false)
	if err != nil {
		return errors.Wrapf(err, "failed to push to remote %s from dir %s", remote, dir)
	}

	log.Logger().Infof("pushed requirements changes to %s", util.ColorInfo(o.GitCloneURL))
	return nil
}

// gitCloneIfRequired if the specified directory is already a git clone then lets just use it
// otherwise lets make a temporary directory and clone the git repository specified
// or if there is none make a new one
func (o *VerifyOptions) gitCloneIfRequired(gitter gits.Gitter) (string, error) {
	gitURL := o.GitCloneURL
	if gitURL == "" {
		return "", util.MissingOption("git-url")
	}
	var err error
	dir := o.Dir
	if dir != "" {
		err = os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create directory %s", dir)
		}
	} else {
		dir, err = ioutil.TempDir("", "helmboot-")
		if err != nil {
			return "", errors.Wrap(err, "failed to create temporary directory")
		}
	}

	log.Logger().Infof("cloning %s to directory %s", util.ColorInfo(gitURL), util.ColorInfo(dir))

	err = gitter.Clone(gitURL, dir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone repository %s to directory: %s", gitURL, dir)
	}
	return dir, nil
}