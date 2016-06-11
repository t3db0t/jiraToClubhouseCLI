package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Jira to Clubhouse"
	app.Usage = "Jira To Clubhouse"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:    "export",
			Aliases: []string{"e"},
			Usage:   "Export Jira XMl into a clubhouse-esque json file",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "in, i",
					Usage: "The Jira XML file you want to read in.",
				},
				cli.IntFlag{
					Name:  "projectId, p",
					Usage: "The project ID you want these items imported for",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "The destination file",
				},
			},
			Action: func(c *cli.Context) error {
				jiraFile := c.String("in")
				exportFile := c.String("out")
				projectId := c.Int("projectId")

				if jiraFile == "" {
					fmt.Println("An input file must be specified.")
					return nil
				}

				if exportFile == "" {
					fmt.Println("An output file must be specified.")
					return nil
				}

				if projectId == 0 {
					fmt.Println("A projectId must be specified.")
					return nil
				}
				err := ExportToJSON(jiraFile, int64(projectId), exportFile)
				if err != nil {
					fmt.Println(err)
					return err
				}
				return nil
			},
		}, {
			Name:    "import",
			Aliases: []string{"i"},
			Usage:   "Import Jira XMl into Clubhouse",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "in, i",
					Usage: "The Jira XML file you want to read in.",
				},
				cli.IntFlag{
					Name:  "projectId, p",
					Usage: "The project ID you want these items imported for",
				},
				cli.StringFlag{
					Name:  "token, t",
					Usage: "Your API token",
				},
			},
			Action: func(c *cli.Context) error {
				jiraFile := c.String("in")
				token := c.String("token")
				projectId := c.Int("projectId")

				if jiraFile == "" {
					fmt.Println("An input file must be specified.")
					return nil
				}

				if token == "" {
					fmt.Println("A token must be specified.")
					return nil
				}

				if projectId == 0 {
					fmt.Println("A projectId must be specified.")
					return nil
				}
				err := UploadToClubhouse(jiraFile, int64(projectId), token)
				if err != nil {
					fmt.Println(err)
					return err
				}
				return nil
			},
		},
	}
	app.Run(os.Args)
}

func GetStoryJSONByteArray(data ClubHouseCreateStory) ([]byte, error) {
	jsonObj, err := json.Marshal(data)
	if err != nil {
		return []byte{}, err
	}
	jsonStr := strings.Replace(string(jsonObj), "\"epic_id\":0,", "", -1)
	return []byte(jsonStr), nil
}

func ExportToJSON(jiraFile string, projectId int64, exportFile string) error {
	export, err := GetDataFromXMLFile(jiraFile)
	if err != nil {
		return err
	}
	data, err := json.Marshal(export.GetDataForClubhouse(projectId))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(exportFile, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func UploadToClubhouse(jiraFile string, projectId int64, token string) error {
	export, err := GetDataFromXMLFile(jiraFile)
	if err != nil {
		return err
	}
	data := export.GetDataForClubhouse(projectId)
	err = SendData(token, data)
	if err != nil {
		return err
	}
	return nil
}

func SendData(token string, data ClubHouseData) error {
	epicMap := make(map[string]int64)
	client := &http.Client{}
	for _, epic := range data.Epics {
		jsonStr, _ := json.Marshal(epic)
		req, err := http.NewRequest("POST", GetUrl("epics", token), bytes.NewBuffer(jsonStr))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode > 299 {
			fmt.Println("response Status:", resp.Status)
			fmt.Println("response Headers:", resp.Header)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		newEpic := ClubHouseEpic{}
		json.Unmarshal(body, &newEpic)
		epicMap[epic.key] = newEpic.Id
	}
	for _, story := range data.Stories {
		if story.epicLink != "" {
			story.EpicId = epicMap[story.epicLink]
		}
		jsonObj, err := GetStoryJSONByteArray(story)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", GetUrl("stories", token), bytes.NewBuffer(jsonObj))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode > 299 {
			fmt.Println("response Status:", resp.Status)
			fmt.Println("response Headers:", resp.Header)
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
		}
	}
	return nil
}

func GetUrl(kind string, token string) string {
	return fmt.Sprintf("%s%s?token=%s", "https://api.clubhouse.io/api/v1/", kind, token)
}

func GetDataFromXMLFile(jiraFile string) (JiraExport, error) {
	xmlFile, err := os.Open(jiraFile)
	if err != nil {
		return JiraExport{}, err
	}
	defer xmlFile.Close()
	XMLData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return JiraExport{}, err
	}
	jiraExport := JiraExport{}
	err = xml.Unmarshal(XMLData, &jiraExport)
	if err != nil {
		return JiraExport{}, err
	}
	return jiraExport, nil
}
