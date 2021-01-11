/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package exec

import (
	"context"
	"fmt"
	"path"

	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
)

type CorruptActionSpec struct {
	spec.BaseExpActionCommandSpec
}

func NewCorruptActionSpec() spec.ExpActionCommandSpec {
	return &CorruptActionSpec{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: commFlags,
			ActionFlags: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:     "percent",
					Desc:     "Corruption percent, must be positive integer without %, for example, --percent 50",
					Required: true,
				},
			},
			ActionExecutor: &NetworkCorruptExecutor{},
			ActionExample: `
# Access to the specified IP request packet is corrupted, 80% of the time
blade create network corrupt --percent 80 --destination-ip 180.101.49.12 --interface eth0`,
			ActionPrograms: []string{TcNetworkBin},
		},
	}
}

func (*CorruptActionSpec) Name() string {
	return "corrupt"
}

func (*CorruptActionSpec) Aliases() []string {
	return []string{}
}

func (*CorruptActionSpec) ShortDesc() string {
	return "Corrupt experiment"
}

func (c *CorruptActionSpec) LongDesc() string {
	if c.ActionLongDesc != "" {
		return c.ActionLongDesc
	}
	return "Corrupt experiment"
}

type NetworkCorruptExecutor struct {
	channel spec.Channel
}

func (ce *NetworkCorruptExecutor) Name() string {
	return "corrupt"
}

func (ce *NetworkCorruptExecutor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {
	commands := []string{"tc", "head"}
	if response, ok := channel.NewLocalChannel().IsAllCommandsAvailable(commands); !ok {
		return response
	}

	if ce.channel == nil {
		util.Errorf(uid, util.GetRunFuncName(), spec.ResponseErr[spec.ChannelNil].ErrInfo)
		return spec.ResponseFail(spec.ChannelNil, spec.ResponseErr[spec.ChannelNil].ErrInfo)
	}
	netInterface := model.ActionFlags["interface"]
	if netInterface == "" {
		util.Errorf(uid, util.GetRunFuncName(), fmt.Sprintf(spec.ResponseErr[spec.ParameterLess].ErrInfo, "interface"))
		return spec.ResponseFailWaitResult(spec.ParameterLess, fmt.Sprintf(spec.ResponseErr[spec.ParameterLess].Err, "interface"),
			fmt.Sprintf(spec.ResponseErr[spec.ParameterLess].ErrInfo, "interface"))
	}
	if _, ok := spec.IsDestroy(ctx); ok {
		return ce.stop(netInterface, ctx)
	} else {
		percent := model.ActionFlags["percent"]
		if percent == "" {
			util.Errorf(uid, util.GetRunFuncName(), fmt.Sprintf(spec.ResponseErr[spec.ParameterLess].ErrInfo, "percent"))
			return spec.ResponseFailWaitResult(spec.ParameterLess, fmt.Sprintf(spec.ResponseErr[spec.ParameterLess].Err, "percent"),
				fmt.Sprintf(spec.ResponseErr[spec.ParameterLess].ErrInfo, "percent"))
		}
		localPort := model.ActionFlags["local-port"]
		remotePort := model.ActionFlags["remote-port"]
		excludePort := model.ActionFlags["exclude-port"]
		destIp := model.ActionFlags["destination-ip"]
		excludeIp := model.ActionFlags["exclude-ip"]
		ignorePeerPort := model.ActionFlags["ignore-peer-port"] == "true"
		force := model.ActionFlags["force"] == "true"
		return ce.start(netInterface, localPort, remotePort, excludePort, destIp, excludeIp, percent, ignorePeerPort, force, ctx)
	}
}

func (ce *NetworkCorruptExecutor) start(netInterface, localPort, remotePort, excludePort, destIp, excludeIp, percent string,
	ignorePeerPort, force bool, ctx context.Context) *spec.Response {
	args := fmt.Sprintf("--start --type corrupt --interface %s --percent %s --debug=%t", netInterface, percent, util.Debug)
	args, err := getCommArgs(localPort, remotePort, excludePort, destIp, excludeIp, args, ignorePeerPort, force)
	if err != nil {
		return spec.ResponseFailWaitResult(spec.ParameterIllegal, err.Error(), err.Error())
	}
	return ce.channel.Run(ctx, path.Join(ce.channel.GetScriptPath(), TcNetworkBin), args)
}

func (ce *NetworkCorruptExecutor) stop(netInterface string, ctx context.Context) *spec.Response {
	return ce.channel.Run(ctx, path.Join(ce.channel.GetScriptPath(), TcNetworkBin),
		fmt.Sprintf("--stop --type corrupt --interface %s --debug=%t", netInterface, util.Debug))
}

func (ce *NetworkCorruptExecutor) SetChannel(channel spec.Channel) {
	ce.channel = channel
}
