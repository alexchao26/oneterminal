package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/spf13/cobra"
)

func makeUpdateCmd(currentVersion string) *cobra.Command {
	var dryRun, yesFlag bool

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Updates oneterminal to latest release",
		Long: `Updates to the latest release using 'go install' as the pkg manager,
so a local Go installation is required`,
		// big ol' function
		Run: func(_ *cobra.Command, _ []string) {
			mostRecentVersion, err := getMostRecentVersion(currentVersion)
			// non-critical error, continue w/ update after logging
			if err != nil {
				fmt.Println(err)
				// for running go install cmd
				mostRecentVersion = "latest"
			} else {
				fmt.Printf("Most recent version is %s\n", mostRecentVersion)
			}

			if mostRecentVersion == "v"+currentVersion {
				fmt.Println("Already up to date, Exiting.")
				return
			}

			if dryRun {
				fmt.Println("Exiting after dry run")
				return
			}

			// Ask user if they want to continue with install
			if !yesFlag {
				var input string
				fmt.Print("Continue with update? (y)es or (n)o: ")
				fmt.Scanln(&input)
				if input != "y" && input != "yes" {
					fmt.Println("Exiting...")
					return
				}
			}

			// will error at c.Run if go is not installed locally
			fmt.Printf("Running `go install github.com/alexchao26/oneterminal@%s`\n", mostRecentVersion)
			c := exec.Command("go", "install", fmt.Sprintf("github.com/alexchao26/oneterminal@%s", mostRecentVersion))
			if err := c.Run(); err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println("Done.")
		},
	}

	updateCmd.Flags().BoolVarP(&dryRun, "dryrun", "d", false, "just check Github for the latest release, do not install")
	updateCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "answer yes to all prompts")

	return updateCmd
}

func getMostRecentVersion(currentVersion string) (string, error) {
	fmt.Println("Checking for the latest release from Github...")
	req, err := http.NewRequest("GET", "https://api.github.com/repos/alexchao26/oneterminal/releases?page=1", nil)
	if err != nil {
		return "", fmt.Errorf("making Github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("making Github request: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad response from Github %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	var githubResp []struct {
		TagName string `json:"tag_name"`
	}
	err = json.NewDecoder(resp.Body).Decode(&githubResp)
	if err != nil {
		return "", fmt.Errorf("decoding Github API Response: %w", err)
	}

	var mostRecentVersion string
	for _, release := range githubResp {
		if release.TagName > mostRecentVersion {
			mostRecentVersion = release.TagName
		}
	}

	if mostRecentVersion == "" {
		return "", fmt.Errorf("didn't find any Github releases")
	}
	return mostRecentVersion, nil
}
