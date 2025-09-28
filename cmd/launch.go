package cmd

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/spf13/cobra"
	"limeal.fr/launchygo/connectors"
	"limeal.fr/launchygo/game/authenticator"
	"limeal.fr/launchygo/game/folder"
	"limeal.fr/launchygo/game/folder/rules"
	"limeal.fr/launchygo/game/launcher"
	"limeal.fr/launchygo/game/profile"
)

var xmx int
var xms int
var javaPath string
var auth string
var mcServer string // If set (add quickPlay=true and quickPlayMultiplayer = <value>)

var launchCmd = &cobra.Command{
	Use:   "launch <game_folder> <uri>",
	Short: "Download and launch minecraft from uri (sftp, files)",
	Long: `Download and launch minecraft from uri (sftp, files, etc.).
  
	Arguments:
  <game_folder>  The path to the game folder to launch.
  <uri>          The uri to the game folder to launch.

  The launch command will download and launch a minecraft game folder on the specified uri.
`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var authAuthenticator authenticator.Authenticator
		var urlOpts *url.URL
		if auth != "" {
			var err error
			authAuthenticator, urlOpts, err = authenticator.FindAuthenticatorFromURI(auth)
			if err != nil {
				panic(err)
			} else {
				fmt.Println("Authentication successful")
			}
		}

		connector := connectors.FindConnectorFromURI(args[1])
		if connector == nil {
			panic("failed to find connector")
		}

		fmt.Println("Connecting to connector")
		fmt.Println("Connector: ", connector.GetURI())
		err := connector.Connect()
		if err != nil {
			fmt.Println("❌ Failed to connect to the connector")
			fmt.Println(err)
			return
		}
		defer connector.Close()

		gameFolder, err := folder.InitGameFolder(connector, args[0])
		if err != nil {
			panic(err)
		}
		gameProfile := profile.NewGameProfile()

		if authAuthenticator != nil {
			switch authAuthenticator.GetType() {
			case authenticator.MICROSOFT:
				gameProfile.AuthenticateWithCode(authAuthenticator, urlOpts.User.Username())
			case authenticator.CUSTOM:
				password, _ := urlOpts.User.Password()
				gameProfile.AuthenticateWithCredentials(authAuthenticator, urlOpts.User.Username(), password)
			}
		}

		gameProfile.SetMemory(xmx, xms)
		launcherInstance := launcher.NewLauncher()

		if javaPath != "" {
			launcherInstance.SetJavaPath(javaPath)
		}

		err = gameFolder.Build(false, nil)
		if err != nil {
			panic(err)
		}

		launcherInstance.SetGameFolder(gameFolder)
		launcherInstance.SetProfile(gameProfile)

		launchOptions := launcher.RunOptions{
			LogOutput: func(msg string) {
				fmt.Println(msg)
			},
		}

		if mcServer != "" {
			launchOptions.GameFeatures = []rules.Feature{
				{
					AKey:  "has_quick_plays_support",
					Flag:  "quickPlayPath",
					Value: filepath.Join(gameFolder.GetPath(), "quickPlay/log.json"),
				},
				{
					AKey:  "is_quick_play_multiplayer",
					Flag:  "quickPlayMultiplayer",
					Value: mcServer,
				},
			}
		}

		launcherInstance.Run(debug, launchOptions)
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
	launchCmd.Flags().IntVarP(&xmx, "Xmx", "x", 4, "The memory to use for the game")
	launchCmd.Flags().IntVarP(&xms, "Xms", "s", 2, "The memory to use for the game")
	launchCmd.Flags().StringVarP(&javaPath, "java", "j", "", "The path to the java executable")
	launchCmd.Flags().StringVarP(&auth, "auth", "a", "", "If you use authentication (available: microsoft, custom) (e.g microsoft://code, custom://username:password@base_url/login_endpoint)")
	launchCmd.Flags().StringVar(&mcServer, "quickPlayMultiplayer", "", "If you want to join a minecraft server (e.g mc.example.com)")
}
