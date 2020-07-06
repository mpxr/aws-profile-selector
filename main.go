package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"strings"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

type credential struct {
	accessKey string
	secretKey string
	error     string
}

type credentials struct {
	credentials map[string]credential
	current     string
}

var creds credentials

func main() {
	onExit := func() {
		fmt.Println("Exiting")
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	load()

	systray.AddSeparator()
	mInfo := systray.AddMenuItem("Info", "Info on GitHub")
	mQuitOrig := systray.AddMenuItem("Quit", "Quit the app")

	for {
		select {
		case <-mQuitOrig.ClickedCh:
			systray.Quit()
		case <-mInfo.ClickedCh:
			open.Run("https://github.com/mpxr/aws-profile-select")
		}
	}
}

func load() {
	fileName := getFileName()
	inp, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	creds.credentials = make(map[string]credential)

	lines := strings.Split(string(inp), "\n")
	for i, line := range lines {
		profileName := strings.Trim(line, " ")
		if strings.Contains(profileName, "[") && strings.Contains(profileName, "]") {
			var cred credential

			if i+1 >= len(lines) || i+2 >= len(lines) {
				cred.error = fmt.Sprintf("[%s] profile is not followed by aws_access_key_id and aws_secret_access_key lines.", profileName)
			} else {
				line1 := strings.Trim(lines[i+1][strings.Index(lines[i+1], "=")+1:], " ")
				line2 := strings.Trim(lines[i+2][strings.Index(lines[i+2], "=")+1:], " ")

				if strings.Contains(lines[i+1], "aws_access_key_id") {
					cred.accessKey = line1
					cred.secretKey = line2
				} else if strings.Contains(lines[i+1], "aws_secret_access_key") {
					cred.accessKey = line2
					cred.secretKey = line1
				} else {
					cred.error = fmt.Sprintf("[%s] profile is not followed by aws_access_key_id and aws_secret_access_key lines.", profileName)
				}
			}

			creds.credentials[profileName] = cred

		}
	}

	def := creds.credentials["[default]"]

	for k, v := range creds.credentials {
		if k != "[default]" && v.accessKey == def.accessKey && v.secretKey == def.secretKey {
			creds.current = k
			break
		}
	}

	delete(creds.credentials, "[default]")

	systray.SetTitle(creds.current)
	systray.SetTooltip("Choose your AWS profile")

	var menuItems = make(map[string]*systray.MenuItem)
	for profile := range creds.credentials {
		c := systray.AddMenuItem(profile, profile)
		menuItems[profile] = c
		// check the current profile when rendering it first
		if profile == creds.current {
			c.Check()
		} else {
			if creds.credentials[profile].error != "" {
				c.Disable()
			}
		}
		go clicked(c, profile, menuItems)
	}
}

func clicked(c *systray.MenuItem, name string, menuItems map[string]*systray.MenuItem) {
	for {
		select {
		case <-c.ClickedCh:
			{
				changeDefaultProfile(name)

				// uncheck all menu items
				for _, v := range menuItems {
					v.Uncheck()
				}

				menuItems[name].Check()
			}
		}
	}
}

func changeDefaultProfile(name string) {
	cred, found := creds.credentials[name]
	if !found {
		log.Fatal(name + " not found in profile")
	}

	updateDefault(cred.accessKey, cred.secretKey)
	systray.SetTitle(name)

}

func getFileName() string {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	fileName := user.HomeDir + "/.aws/credentials"
	return fileName
}

func updateDefault(accessKey string, secretKey string) {
	lines := readCredentials()
	for i, line := range lines {
		if strings.Contains(line, "[default]") {
			lines[i+1] = fmt.Sprintf("aws_access_key_id = %s", accessKey)
			lines[i+2] = fmt.Sprintf("aws_secret_access_key = %s", secretKey)
			break
		}
	}

	output := strings.Join(lines, "\n")

	fileName := getFileName()
	err := ioutil.WriteFile(fileName, []byte(output), 0644)

	if err != nil {
		log.Fatalln(err)
	}
}

func readCredentials() []string {
	fileName := getFileName()

	var lines []string

	inp, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
		return lines
	}

	lines = strings.Split(string(inp), "\n")

	return lines
}
