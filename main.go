package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

func main() {
	onExit := func() {
		fmt.Println("Exiting")
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	profiles := readProfileNames()

	systray.SetTitle("AWS Profile Select")
	systray.SetTooltip("Choose your AWS profile")

	for _, profile := range profiles {
		profileName := profile[1 : len(profile)-1]
		c := systray.AddMenuItem(profileName, profileName)
		go clicked(c, profileName)
	}

	systray.AddSeparator()
	mInfo := systray.AddMenuItem("Info", "Info")
	mQuitOrig := systray.AddMenuItem("Quit", "Quit the app")
	go func() {
		<-mQuitOrig.ClickedCh
		systray.Quit()
	}()
	go func() {
		<-mInfo.ClickedCh
		open.Run("https://github.com/mpxr/aws-profile-select")
	}()
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

func getProfileFileName() (*(os.File), string) {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	fileName := user.HomeDir + "/.aws/credentials"

	fileToRead, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	return fileToRead, fileName
}

func readProfileNames() []string {
	fileToRead, _ := getProfileFileName()
	defer fileToRead.Close()

	var profiles []string
	scanner := bufio.NewScanner(fileToRead)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "[default]") && strings.Contains(line, "[") && strings.Contains(line, "]") {
			profiles = append(profiles, line)
		}
	}

	if error := scanner.Err(); error != nil {
		log.Fatal(error)
	}

	return profiles
}

func getSecret(name string) (string, string) {
	fileToRead, _ := getProfileFileName()
	defer fileToRead.Close()

	accessKeyLine, secretKeyLine, err := func() (string, string, error) {
		found := false
		var accessKey string
		var secretKey string
		scanner := bufio.NewScanner(fileToRead)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), fmt.Sprintf("[%s]", name)) {
				found = true
			} else if found && (accessKey == "" || secretKey == "") {
				if strings.Contains(scanner.Text(), "aws_access_key_id") {
					accessKey = scanner.Text()
				} else if strings.Contains(scanner.Text(), "aws_secret_access_key") {
					secretKey = scanner.Text()
				}
			}
			if found && accessKey != "" && secretKey != "" {
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
	fileToRead, fileName := getProfileFileName()
	fileToRead.Close()

	inp, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
		return
	}

	lines := strings.Split(string(inp), "\n")

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
		log.Fatalln(err)
	}
}
