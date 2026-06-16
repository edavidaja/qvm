package cmd

import (
	"fmt"

	"github.com/dpastoor/qvm/internal/config"
	"github.com/dpastoor/qvm/internal/edition"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type pathVersionsCmd struct {
	cmd  *cobra.Command
	opts pathVersionsOpts
}

type pathVersionsOpts struct {
	q2 bool
}

func newPathVersions(pathVersionsOpts pathVersionsOpts) error {
	e := edition.FromFlag(pathVersionsOpts.q2)
	fmt.Println(config.GetPathToVersionsDir(e))
	return nil
}

func setPathVersionsOpts(pathVersionsOpts *pathVersionsOpts) {
	pathVersionsOpts.q2 = viper.GetBool("path-versions-q2")
}

func (opts *pathVersionsOpts) Validate() error {
	return nil
}

func newPathVersionsCmd() *pathVersionsCmd {
	root := &pathVersionsCmd{opts: pathVersionsOpts{}}

	cmd := &cobra.Command{
		Use:   "versions",
		Short: "get the path to where the various quarto versions are stored",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			setPathVersionsOpts(&root.opts)
			if err := root.opts.Validate(); err != nil {
				return err
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			//TODO: Add your logic to gather config to pass code here
			log.WithField("opts", fmt.Sprintf("%+v", root.opts)).Trace("pathVersions-opts")
			if err := newPathVersions(root.opts); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().Bool("q2", false, "manage Quarto v2 (q2) instead of classic quarto")
	viper.BindPFlag("path-versions-q2", cmd.Flags().Lookup("q2"))
	root.cmd = cmd
	return root
}
