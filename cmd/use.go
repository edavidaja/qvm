package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/coreos/go-semver/semver"
	"github.com/dpastoor/qvm/internal/config"
	"github.com/dpastoor/qvm/internal/edition"
	"github.com/dpastoor/qvm/internal/gh"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

type useCmd struct {
	cmd  *cobra.Command
	opts useOpts
}

type useOpts struct {
	install bool
	q2      bool
}

func newUse(useOpts useOpts, version string) error {
	e := edition.FromFlag(useOpts.q2)
	iv, err := config.GetInstalledVersions(e)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	versions := maps.Keys(iv)
	var semVersions semver.Versions
	// sorting has some issues given how the character values will present individually
	// for example a high value double digit patch version will be sorted before a lower
	// value triple digit patch version. For example, sorting shows ordering like:
	// v1.2.89 v1.2.237 v1.2.112 v1.1.84 v1.1.251 v1.1.189
	// where .89 is > 237
	// using go-semver this works
	// as will get 1.2.237 1.2.112 1.2.89 1.1.251 1.1.189 1.1.168 1.1.84
	for _, v := range versions {
		ver, err := semver.NewVersion(strings.TrimPrefix(v, "v"))
		if err != nil {
			// we're just going to warn rather than error right now in case some
			// releases end up not following semver and would rather the tool not blow up
			log.Errorf("could not parse semver value for %s with err %s\n ", v, err)
			continue
		}
		semVersions = append(semVersions, ver)
	}
	sort.Sort(sort.Reverse(semVersions))
	// note this could be a bug if ever we do get nonparseable versions thrown out above
	// will cross that bridge if we get there
	for i, v := range semVersions {
		versions[i] = "v" + v.String()
	}
	// convert back to string for later options
	if len(iv) == 0 && !useOpts.install {
		return errors.New("no installed versions found, please install a version first")
	}
	client := gh.NewClient(os.Getenv("GITHUB_PAT"))
	if version == "release" {
		latestRelease, err := gh.GetLatestRelease(client, e.Owner, e.Repo)
		if err != nil {
			return err
		}
		version = latestRelease.GetTagName()
	}
	if version == "latest" {
		if useOpts.install {
			// this will install further down if the version isn't already installed
			repo, err := gh.GetReleases(client, e.Owner, e.Repo, 1, "")
			if err != nil {
				return err
			}
			version = repo[0].GetTagName()
		} else {
			version = versions[0]
		}
		// add back the v we trimmed for semver
	}
	if version == "" {
		// not worried about an error here as an active version of
		// empty string just won't match any description below
		activeVersion, _ := config.GetActiveVersion(e)
		err = survey.AskOne(&survey.Select{
			Message: "Which version do you want to use?",
			Options: versions,
			Description: func(value string, index int) string {
				if value == activeVersion {
					return "**active**"
				}
				return ""
			},
		}, &version, survey.WithPageSize(10))
		if err != nil {
			return err
		}
	}
	quartopath, ok := iv[version]
	if !ok {
		if useOpts.install {
			err, version = newInstall(installOpts{progress: true, q2: useOpts.q2}, version)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("version %s not found", version)
		}
	}
	err = os.MkdirAll(config.GetPathToActiveBinDir(), 0755)
	if err != nil {
		return err
	}
	activeExe := config.GetPathToActiveExe(e)
	err = os.Remove(activeExe)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = os.Symlink(
		quartopath,
		activeExe,
	)
	if err != nil {
		return err
	}
	log.Infof("now using %s version: %s\n", e.Label(), version)
	return nil
}

func setUseOpts(useOpts *useOpts) {
	useOpts.install = viper.GetBool("install")
	useOpts.q2 = viper.GetBool("use-q2")
}

func (opts *useOpts) Validate() error {
	return nil
}

func newUseCmd() *useCmd {
	root := &useCmd{opts: useOpts{}}

	cmd := &cobra.Command{
		Use:   "use",
		Short: "use a version of quarto",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			setUseOpts(&root.opts)
			if err := root.opts.Validate(); err != nil {
				return err
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			//TODO: Add your logic to gather config to pass code here
			log.WithField("opts", fmt.Sprintf("%+v", root.opts)).Trace("use-opts")
			var version string
			if len(args) > 0 {
				version = args[0]
			}
			if err := newUse(root.opts, version); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().Bool("install", false, "install the version if not already installed")
	viper.BindPFlag("install", cmd.Flags().Lookup("install"))
	cmd.Flags().Bool("q2", false, "manage Quarto v2 (q2) instead of classic quarto")
	viper.BindPFlag("use-q2", cmd.Flags().Lookup("q2"))
	root.cmd = cmd
	return root
}
