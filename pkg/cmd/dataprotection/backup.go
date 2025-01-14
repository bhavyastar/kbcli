/*
Copyright (C) 2022-2023 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package dataprotection

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/apecloud/kbcli/pkg/action"
	"github.com/apecloud/kbcli/pkg/cmd/cluster"
	"github.com/apecloud/kbcli/pkg/printer"
	"github.com/apecloud/kbcli/pkg/types"
	"github.com/apecloud/kbcli/pkg/util"
)

var (
	createBackupExample = templates.Examples(`
		# Create a backup for the cluster, use the default backup policy and volume snapshot backup method
		kbcli dp backup mybackup --cluster mycluster

		# create a backup with a specified method, run "kbcli cluster desc-backup-policy mycluster" to show supported backup methods
		kbcli dp backup mybackup --cluster mycluster --method mymethod

		# create a backup with specified backup policy, run "kbcli cluster list-backup-policy mycluster" to show the cluster supported backup policies
		kbcli dp backup mybackup --cluster mycluster --policy mypolicy

		# create a backup from a parent backup
		kbcli dp backup mybackup --cluster mycluster --parent-backup myparentbackup
	`)

	deleteBackupExample = templates.Examples(`
		# delete a backup
		kbcli dp delete-backup mybackup
	`)

	describeBackupExample = templates.Examples(`
		# describe a backup
		kbcli dp describe-backup mybackup
	`)

	listBackupExample = templates.Examples(`
		# list all backups
		kbcli dp list-backups

		# list all backups of specified cluster
		kbcli dp list-backups --cluster mycluster
	`)
)

func newBackupCommand(f cmdutil.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	customOutPut := func(opt *action.CreateOptions) {
		output := fmt.Sprintf("Backup %s created successfully, you can view the progress:", opt.Name)
		printer.PrintLine(output)
		nextLine := fmt.Sprintf("\tkbcli dp list-backups %s -n %s", opt.Name, opt.Namespace)
		printer.PrintLine(nextLine)
	}

	clusterName := ""

	o := &cluster.CreateBackupOptions{
		CreateOptions: action.CreateOptions{
			IOStreams:       streams,
			Factory:         f,
			GVR:             types.OpsGVR(),
			CueTemplateName: "opsrequest_template.cue",
			CustomOutPut:    customOutPut,
		},
	}
	o.CreateOptions.Options = o

	cmd := &cobra.Command{
		Use:     "backup NAME",
		Short:   "Create a backup for the cluster.",
		Example: createBackupExample,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				o.BackupSpec.BackupName = args[0]
			}
			if clusterName != "" {
				o.Args = []string{clusterName}
			}
			cmdutil.BehaviorOnFatal(printer.FatalWithRedColor)
			cmdutil.CheckErr(o.CompleteBackup())
			cmdutil.CheckErr(o.Validate())
			cmdutil.CheckErr(o.Run())
		},
	}

	cmd.Flags().StringVar(&o.BackupSpec.BackupMethod, "method", "", "Backup methods are defined in backup policy (required), if only one backup method in backup policy, use it as default backup method, if multiple backup methods in backup policy, use method which volume snapshot is true as default backup method")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name")
	cmd.Flags().StringVar(&o.BackupSpec.BackupPolicyName, "policy", "", "Backup policy name, if not specified, use the cluster default backup policy")
	cmd.Flags().StringVar(&o.BackupSpec.DeletionPolicy, "deletion-policy", "Delete", "Deletion policy for backup, determine whether the backup content in backup repo will be deleted after the backup is deleted, supported values: [Delete, Retain]")
	cmd.Flags().StringVar(&o.BackupSpec.RetentionPeriod, "retention-period", "", "Retention period for backup, supported values: [1y, 1mo, 1d, 1h, 1m] or combine them [1y1mo1d1h1m], if not specified, the backup will not be automatically deleted, you need to manually delete it.")
	cmd.Flags().StringVar(&o.BackupSpec.ParentBackupName, "parent-backup", "", "Parent backup name, used for incremental backup")
	util.RegisterClusterCompletionFunc(cmd, f)
	o.RegisterBackupFlagCompletionFunc(cmd, f)

	return cmd
}

func newBackupDeleteCommand(f cmdutil.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	o := action.NewDeleteOptions(f, streams, types.BackupGVR())
	clusterName := ""
	cmd := &cobra.Command{
		Use:               "delete-backup",
		Short:             "Delete a backup.",
		Example:           deleteBackupExample,
		ValidArgsFunction: util.ResourceNameCompletionFunc(f, types.BackupGVR()),
		Run: func(cmd *cobra.Command, args []string) {
			o.Names = args
			cmdutil.BehaviorOnFatal(printer.FatalWithRedColor)
			util.CheckErr(completeForDeleteBackup(o, clusterName))
			util.CheckErr(o.Run())
		},
	}

	o.AddFlags(cmd)
	cmd.Flags().StringVar(&clusterName, "cluster", "", "The cluster name.")
	util.RegisterClusterCompletionFunc(cmd, f)

	return cmd
}

func completeForDeleteBackup(o *action.DeleteOptions, cluster string) error {
	if o.Force && len(o.Names) == 0 {
		if cluster == "" {
			return fmt.Errorf("must give a backup name or cluster name")
		}
		o.LabelSelector = util.BuildLabelSelectorByNames(o.LabelSelector, []string{cluster})
	}
	return nil
}

func newBackupDescribeCommand(f cmdutil.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	o := cluster.DescribeBackupOptions{
		Factory:   f,
		IOStreams: streams,
		Gvr:       types.BackupGVR(),
	}
	cmd := &cobra.Command{
		Use:               "describe-backup NAME",
		Short:             "Describe a backup",
		Aliases:           []string{"desc-backup"},
		ValidArgsFunction: util.ResourceNameCompletionFunc(f, types.BackupGVR()),
		Example:           describeBackupExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.BehaviorOnFatal(printer.FatalWithRedColor)
			util.CheckErr(o.Complete(args))
			util.CheckErr(o.Run())
		},
	}
	return cmd
}

func newListBackupCommand(f cmdutil.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	o := &cluster.ListBackupOptions{ListOptions: action.NewListOptions(f, streams, types.BackupGVR())}
	clusterName := ""
	cmd := &cobra.Command{
		Use:               "list-backups",
		Short:             "List backups.",
		Aliases:           []string{"ls-backups"},
		Example:           listBackupExample,
		ValidArgsFunction: util.ResourceNameCompletionFunc(f, o.GVR),
		Run: func(cmd *cobra.Command, args []string) {
			if clusterName != "" {
				o.LabelSelector = util.BuildLabelSelectorByNames(o.LabelSelector, []string{clusterName})
			}
			o.Names = args
			cmdutil.BehaviorOnFatal(printer.FatalWithRedColor)
			cmdutil.CheckErr(o.Complete())
			cmdutil.CheckErr(cluster.PrintBackupList(*o))
		},
	}
	o.AddFlags(cmd, true)
	cmd.Flags().StringVar(&clusterName, "cluster", "", "List backups in the specified cluster")
	util.RegisterClusterCompletionFunc(cmd, f)

	return cmd
}
