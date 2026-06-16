package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dpastoor/qvm/internal/config"
	"github.com/dpastoor/qvm/internal/edition"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type rmCmd struct {
	cmd  *cobra.Command
	opts rmOpts
}

type rmOpts struct {
	q2 bool
}

func newRm(rmOpts rmOpts, versions []string) error {
	e := edition.FromFlag(rmOpts.q2)
	for _, v := range versions {
		if v == "latest" || v == "release" {
			log.Fatalf("qvm rm does not support latest/release terms for now, please use explicit versions only, such as v1.1.251\n")
		}
	}
	activeVersion, err := config.GetActiveVersion(e)
	if err != nil {
		return err
	}
	allVersions, err := config.GetInstalledVersions(e)
	if err != nil {
		return err
	}
	for _, v := range versions {
		if _, ok := allVersions[v]; !ok {
			log.Warnf("version %s is not installed, so not removing\n", v)
			continue
		}
		log.Tracef("about to remove %s\n", v)
		if v == activeVersion {
			log.Warnf("removing active version %s, be sure to set a new active version\n", v)
		}
		err := os.RemoveAll(filepath.Join(config.GetPathToVersionsDir(e), v))
		if err != nil {
			return err
		}
		log.Infof("removed version %s\n", v)
	}
	return nil
}

func setRmOpts(rmOpts *rmOpts) {
	rmOpts.q2 = viper.GetBool("rm-q2")
}

func (opts *rmOpts) Validate() error {
	return nil
}

func newRmCmd() *rmCmd {
	root := &rmCmd{opts: rmOpts{}}

	cmd := &cobra.Command{
		Use:   "rm",
		Short: "remove a version of quarto",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			setRmOpts(&root.opts)
			if err := root.opts.Validate(); err != nil {
				return err
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			//TODO: Add your logic to gather config to pass code here
			log.WithField("opts", fmt.Sprintf("%+v", root.opts)).Trace("rm-opts")
			if err := newRm(root.opts, args); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().Bool("q2", false, "manage Quarto v2 (q2) instead of classic quarto")
	viper.BindPFlag("rm-q2", cmd.Flags().Lookup("q2"))
	root.cmd = cmd
	return root
}
