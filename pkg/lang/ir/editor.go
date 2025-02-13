// Copyright 2022 The envd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ir

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/moby/buildkit/client/llb"

	"github.com/tensorchord/envd/pkg/editor/vscode"
	"github.com/tensorchord/envd/pkg/flag"
	"github.com/tensorchord/envd/pkg/progress/compileui"
)

func (g Graph) compileVSCode() (*llb.State, error) {
	if len(g.VSCodePlugins) == 0 {
		return nil, nil
	}
	inputs := []llb.State{}
	for _, p := range g.VSCodePlugins {
		vscodeClient, err := vscode.NewClient(vscode.MarketplaceVendorOpenVSX)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create vscode client")
		}
		g.Writer.LogVSCodePlugin(p, compileui.ActionStart, false)
		if cached, err := vscodeClient.DownloadOrCache(p); err != nil {
			return nil, err
		} else {
			g.Writer.LogVSCodePlugin(p, compileui.ActionEnd, cached)
		}
		ext := llb.Scratch().File(llb.Copy(llb.Local(flag.FlagCacheDir),
			vscodeClient.PluginPath(p),
			"/home/envd/.vscode-server/extensions/"+p.String(),
			&llb.CopyInfo{
				CreateDestPath: true,
			}, llb.WithUIDGID(g.uid, g.gid)),
			llb.WithCustomNamef("install vscode plugin %s", p.String()))
		inputs = append(inputs, ext)
	}
	layer := llb.Merge(inputs, llb.WithCustomName("merging plugins for vscode"))
	return &layer, nil
}

func (g *Graph) compileJupyter() error {
	if g.JupyterConfig != nil {
		g.PyPIPackages = append(g.PyPIPackages, "jupyter")
		switch g.Language.Name {
		case "python":
			return nil
		default:
			return errors.Newf("Jupyter is not supported in %s yet", g.Language.Name)
		}
	}
	return nil
}

func (g Graph) generateJupyterCommand(workingDir string) []string {
	if g.JupyterConfig == nil {
		return nil
	}

	var cmd []string
	// Use python in conda env.
	if g.CondaEnabled() {
		cmd = append(cmd, "/opt/conda/bin/python3")
	} else {
		cmd = append(cmd, "python3")
	}

	cmd = append(cmd, []string{
		"-m", "notebook",
		"--ip", "0.0.0.0", "--notebook-dir", workingDir,
	}...)

	if g.JupyterConfig.Password != "" {
		cmd = append(cmd, "--NotebookApp.password", g.JupyterConfig.Password,
			"--NotebookApp.token", "''")
	} else {
		cmd = append(cmd, "--NotebookApp.password", "''",
			"--NotebookApp.token", "''")
	}
	if g.JupyterConfig.Port != 0 {
		p := strconv.Itoa(int(g.JupyterConfig.Port))
		cmd = append(cmd, "--port", p)
	}
	return cmd
}

func (g Graph) generateRStudioCommand(workingDir string) []string {
	if g.RStudioServerConfig == nil {
		return nil
	}

	return []string{
		// TODO(gaocegege): Remove root permission here.
		"sudo",
		"/usr/lib/rstudio-server/bin/rserver",
		// TODO(gaocegege): Support working dir.
	}
}
