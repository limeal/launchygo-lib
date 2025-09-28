package launcher

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"limeal.fr/launchygo/game/folder/generator/manifests"
	"limeal.fr/launchygo/game/folder/rules"
)

/////////////////////////////////////////////////////////////////////
// Parse and format args
/////////////////////////////////////////////////////////////////////

type LauncherArgumentParser struct {
	launcher *Launcher
}

func NewLauncherArgumentParser(launcher *Launcher) *LauncherArgumentParser {
	return &LauncherArgumentParser{
		launcher: launcher,
	}
}

func (g *LauncherArgumentParser) formatArg(arg string, features ...rules.Feature) string {
	placeholders := make(map[string]string)
	placeholders["auth_player_name"] = g.launcher.profile.Username
	placeholders["version_name"] = g.launcher.gameFolder.GetVersion()
	placeholders["game_directory"] = g.launcher.gameFolder.GetPath()
	placeholders["assets_root"] = filepath.Join(g.launcher.gameFolder.GetPath(), "assets")
	placeholders["assets_index_name"] = g.launcher.gameFolder.GetAssetIndex()

	if g.launcher.profile.UUID != nil {
		placeholders["auth_uuid"] = *g.launcher.profile.UUID
	} else {
		placeholders["auth_uuid"] = "00000000-0000-0000-0000-000000000000"
	}

	placeholders["auth_access_token"] = g.launcher.profile.Token
	placeholders["clientid"] = "0"
	placeholders["auth_xuid"] = "0"
	placeholders["user_type"] = g.launcher.profile.UserType
	placeholders["version_type"] = "release"
	placeholders["resolution_width"] = "1280"
	placeholders["resolution_height"] = "720"
	placeholders["natives_directory"] = filepath.Join(g.launcher.gameFolder.GetPath(), "natives")
	placeholders["launcher_name"] = "launchygo"
	placeholders["launcher_version"] = "1.0.0"

	if len(features) > 0 {
		for _, feature := range features {
			placeholders[feature.Flag] = feature.Value
		}
	}

	cp, err := g.launcher.gameFolder.GetCP()
	if err != nil {
		panic(err)
	}
	placeholders["classpath"] = cp

	// ${placeholder}
	for placeholder, value := range placeholders {
		arg = strings.ReplaceAll(arg, "${"+placeholder+"}", value)
	}

	return arg
}

func (g *LauncherArgumentParser) parseAndFormatArgs(args []any, features ...rules.Feature) []string {

	// regroup them in a map
	formattedArgs := []string{}
	for _, arg := range args {
		if strArg, ok := arg.(string); ok {
			formattedArgs = append(formattedArgs, g.formatArg(strArg, features...))
		} else if ruleArg, ok := arg.(map[string]any); ok {
			values := []string{}
			switch v := ruleArg["value"].(type) {
			case string:
				values = append(values, g.formatArg(v, features...))
			case []interface{}:
				for _, v := range v {
					if str, ok := v.(string); ok {
						values = append(values, g.formatArg(str, features...))
					}
				}
			}

			// json marshal, unmarshal the ruleList
			ruleList := []manifests.Rule{}
			bytes, err := json.Marshal(ruleArg["rules"])
			if err != nil {
				continue
			}
			json.Unmarshal(bytes, &ruleList)

			if rules.ShouldInclude(ruleList, rules.DetectEnv()) {
				formattedArgs = append(formattedArgs, values...)
			} else if ok := rules.ShouldIncludeFeatures(ruleList, features...); ok {
				formattedArgs = append(formattedArgs, values...)
			}

		}
	}

	return formattedArgs
}
