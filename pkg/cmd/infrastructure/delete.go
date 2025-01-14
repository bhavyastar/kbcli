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

package infrastructure

import (
	"fmt"

	"github.com/kubesphere/kubekey/v3/cmd/kk/pkg/common"
	"github.com/kubesphere/kubekey/v3/cmd/kk/pkg/core/connector"
	"github.com/kubesphere/kubekey/v3/cmd/kk/pkg/core/pipeline"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/apecloud/kubeblocks/pkg/gotemplate"

	"github.com/apecloud/kbcli/pkg/util"
)

type deleteOptions struct {
	clusterOptions

	deleteCRI bool
	debug     bool
}

var deleteExamples = templates.Examples(`
	# delete kubernetes cluster with specified config yaml
	kbcli infra delete -c cluster.yaml
`)

func (o *deleteOptions) Run() error {
	o.Cluster.Kubernetes.AutoDefaultFill()
	cluster, err := createClusterWithOptions(&gotemplate.TplValues{
		builtinClusterNameObject:    o.clusterName,
		builtinClusterVersionObject: "0.0.0",
		builtinUserObject:           o.User,
		builtinHostsObject:          o.Nodes,
		builtinTimeoutObject:        o.timeout,
		builtinKubernetesObject:     o.Cluster.Kubernetes,
		builtinRoleGroupsObject: gotemplate.TplValues{
			common.ETCD:   o.RoleGroup.ETCD,
			common.Master: o.RoleGroup.Master,
			common.Worker: o.RoleGroup.Worker,
		},
	})
	if err != nil {
		return err
	}

	yes, err := o.confirm(fmt.Sprintf("delete kubernetes: %s", o.clusterName))
	if err != nil {
		return err
	}
	if !yes {
		return nil
	}

	runtime := &common.KubeRuntime{
		BaseRuntime: connector.NewBaseRuntime(o.clusterName, connector.NewDialer(), o.debug, false),
		Cluster:     cluster,
		ClusterName: o.clusterName,
	}
	syncClusterNodeRole(cluster, runtime)

	pipeline := pipeline.Pipeline{
		Name:    "DeleteCluster",
		Modules: NewDeletePipeline(o),
		Runtime: runtime,
	}
	if err := pipeline.Start(); err != nil {
		return err
	}
	fmt.Fprintf(o.IOStreams.Out, "Kubernetes deletion is complete.\n\n")
	return nil
}

func (o *deleteOptions) buildDeleteInfraFlags(cmd *cobra.Command) {
	buildCommonFlags(cmd, &o.clusterOptions)
	cmd.Flags().BoolVarP(&o.debug, "debug", "", false, "set debug mode")
	cmd.Flags().BoolVarP(&o.deleteCRI, "delete-cri", "", false, "delete cri")
}

func NewDeleteKubernetesCmd(streams genericiooptions.IOStreams) *cobra.Command {
	o := &deleteOptions{
		clusterOptions: clusterOptions{
			IOStreams: streams,
		}}
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "delete kubernetes cluster.",
		Example: deleteExamples,
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(o.Complete())
			util.CheckErr(o.Validate())
			util.CheckErr(o.Run())
		},
	}
	o.buildDeleteInfraFlags(cmd)
	return cmd
}
