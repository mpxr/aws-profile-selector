package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"sort"
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
			open.Run("https://github.com/mpxr/aws-profile-selector")
		}
	}
}

func load() {
	fileName, err := getFileName()
	if err != nil {
		log.Fatal(err)
		return
	}

	inp, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
		return
	}

	// save profile name and credentials in a struct
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

	// save the currently set profile in the struct
	def := creds.credentials["[default]"]
	for k, v := range creds.credentials {
		if k != "[default]" && v.accessKey == def.accessKey && v.secretKey == def.secretKey {
			creds.current = k
			break
		}
	}
	delete(creds.credentials, "[default]")

	// sort the profile names in alphabetical order
	var keys = make([]string, 0, len(creds.credentials))
	for k := range creds.credentials {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	systray.SetTitle(creds.current)
	systray.SetTooltip("Choose your AWS profile")

	var menuItems = make(map[string]*systray.MenuItem)
	for _, profile := range keys {
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
				ok := changeDefaultProfile(name)
				if ok {
					// uncheck all menu items
					for _, v := range menuItems {
						v.Uncheck()
					}

					menuItems[name].Check()
				}
			}
		}
	}
}

func changeDefaultProfile(name string) bool {
	cred, found := creds.credentials[name]
	if !found {
		log.Fatal(name + " not found in profile")
		return false
	}

	ok := updateDefault(cred.accessKey, cred.secretKey)
	if ok {
		systray.SetTitle(name)
	}
	return ok
}

func getFileName() (string, error) {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
		return "", errors.New("Cannot retrieve current user")
	}

	fileName := user.HomeDir + "/.aws/credentials"
	return fileName, nil
}

func updateDefault(accessKey string, secretKey string) bool {
	fileName, err := getFileName()
	if err != nil {
		return false
	}

	lines := readCredentials(fileName)
	for i, line := range lines {
		if strings.Contains(line, "[default]") {
			lines[i+1] = fmt.Sprintf("aws_access_key_id = %s", accessKey)
			lines[i+2] = fmt.Sprintf("aws_secret_access_key = %s", secretKey)
			break
		}
	}

	output := strings.Join(lines, "\n")

	err = ioutil.WriteFile(fileName, []byte(output), 0644)

	if err != nil {
		log.Fatal(err)
		return false
	}

	return true
}

func readCredentials(fileName string) []string {
	var lines []string

	inp, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
		return lines
	}

	lines = strings.Split(string(inp), "\n")

	return lines
}
