package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"strings"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

var lines []string

func main() {
	onExit := func() {
		fmt.Println("Exiting")
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	lines = readCredentials()
	c1 := make(chan []string)
	go readProfileNames(c1)

	systray.SetTitle("AWS Profile Select")
	systray.SetTooltip("Choose your AWS profile")

	profiles := <-c1

	for _, profile := range profiles {
		profileName := profile[1 : len(profile)-1]
		c := systray.AddMenuItem(profileName, profileName)
		go clicked(c, profileName)
	}

	systray.AddSeparator()
	mInfo := systray.AddMenuItem("Info", "Info")
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

func readProfileNames(c chan []string) {
	var profiles []string

	for _, line := range lines {
		if !strings.Contains(line, "[default]") && strings.Contains(line, "[") && strings.Contains(line, "]") {
			profiles = append(profiles, line)
		}
	}

	c <- profiles
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

func clicked(c *systray.MenuItem, name string) {
	for {
		<-c.ClickedCh
		changeDefaultProfile(name)
	}
}

func changeDefaultProfile(name string) {
	// find [<name>] in credentials
	// read the secrets
	accessKey, secretKey := getSecret(name)
	// find [default] in credentials
	// update [default] lines
	updateDefault(accessKey, secretKey)
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

func getSecret(name string) (string, string) {
	accessKeyLine, secretKeyLine, err := func() (string, string, error) {
		var accessKey string
		var secretKey string

		for i, line := range lines {
			if strings.Contains(line, fmt.Sprintf("[%s]", name)) {
				if strings.Contains(lines[i+1], "aws_access_key_id") {
					accessKey = lines[i+1]
					secretKey = lines[i+2]
				} else {
					accessKey = lines[i+2]
					secretKey = lines[i+1]
				}
				return accessKey, secretKey, nil
			}
		}

		return "", "", errors.New("not found")
	}()

	if err != nil {
		log.Fatal("Profile name not found: ", name)
		panic(err)
	}

	accessKey := strings.Trim(accessKeyLine[strings.Index(accessKeyLine, "=")+1:], " ")
	secretKey := strings.Trim(secretKeyLine[strings.Index(secretKeyLine, "=")+1:], " ")

	return accessKey, secretKey
}

func updateDefault(accessKey string, secretKey string) {
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
