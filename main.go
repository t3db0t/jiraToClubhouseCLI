package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

type userMap struct {
	JiraUsername	string
	CHProjectID	int
	CHID		string
}

func main() {
	app := cli.NewApp()
	app.Name = "Jira to Clubhouse"
	app.Usage = "Jira To Clubhouse"
	app.Version = "0.0.4"
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
				cli.StringFlag{
					Name: "map, m",
					Usage: "The JSON file containing user mappings",
				},
				cli.StringFlag{
					Name:  "out, o",
					Usage: "The destination file",
				},
			},
			Action: func(c *cli.Context) error {
				jiraFile := c.String("in")
				exportFile := c.String("out")
				mapFile := c.String("map")

				if jiraFile == "" {
					fmt.Println("An input file must be specified.")
					return nil
				}

				if exportFile == "" {
					fmt.Println("An output file must be specified.")
					return nil
				}

				if mapFile == "" {
					fmt.Println("A user map JSON file must be specified.")
					return nil
				}

				userMaps, err := GetUserMap(mapFile)
				if err != nil {
					fmt.Println(err)
					return err
				}

				err = ExportToJSON(jiraFile, userMaps, exportFile)
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
				cli.StringFlag{
					Name: "map, m",
					Usage: "The JSON file containing user mappings",
				},
				cli.StringFlag{
					Name:  "token, t",
					Usage: "Your API token",
				},
				cli.BoolFlag{
					Name:  "test, T",
					Hidden: false,
					Usage: "Test mode: Does not execute remote requests",
				},
			},
			Action: func(c *cli.Context) error {
				jiraFile := c.String("in")
				token := c.String("token")
				mapFile := c.String("map")
				testMode := c.Bool("test")

				if jiraFile == "" {
					fmt.Println("An input XML file must be specified.")
					return nil
				}

				if token == "" && !testMode {
					fmt.Println("A token must be specified.")
					return nil
				}

				if mapFile == "" {
					fmt.Println("A user map JSON file must be specified.")
					return nil
				}

				userMaps, err := GetUserMap(mapFile)
				if err != nil {
					fmt.Println(err)
					return err
				}

				err = UploadToClubhouse(jiraFile, userMaps, token, testMode)
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

func GetUserMap(mapFile string) ([]userMap, error){
	jsonFile, err := os.Open(mapFile)
	if err != nil {
		return []userMap{}, err
	}

	defer jsonFile.Close()
	JSONData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return []userMap{}, err
	}

	// userMaps := []userMap
	var userMaps []userMap
	err = json.Unmarshal(JSONData, &userMaps)
	if err != nil {
		return []userMap{}, err
	}

	return userMaps, nil
}

// ExportToJSON will import the XML and then export the data to the file specified.
func ExportToJSON(jiraFile string, userMaps []userMap, exportFile string) error {
	export, err := GetDataFromXMLFile(jiraFile)
	if err != nil {
		return err
	}
	data, err := json.Marshal(export.GetDataForClubhouse(userMaps))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(exportFile, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// UploadToClubhouse will import the XML, and upload it to Clubhouse
func UploadToClubhouse(jiraFile string, userMaps []userMap, token string, testMode bool) error {
	export, err := GetDataFromXMLFile(jiraFile)
	if err != nil {
		return err
	}
	data := export.GetDataForClubhouse(userMaps)
	fmt.Printf("Found %d epics and %d stories.\n\n", len(data.Epics), len(data.Stories))
	
	if !testMode{
		fmt.Println("Sending data to Clubhouse...")
		err = SendData(token, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendData will send the data to Clubhouse
func SendData(token string, data ClubHouseData) error {
	// epicMap is used to get the return from the submitting of the ClubHouseCreateEpic to get the ID created by the API so stories can be mapped to the correct epic.
	epicMap := make(map[string]int64)

	client := &http.Client{}

	for _, epic := range data.Epics {
		jsonStr, _ := json.Marshal(epic)
		req, err := http.NewRequest("POST", GetURL("epics", token), bytes.NewBuffer(jsonStr))
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
		epicMap[epic.key] = newEpic.ID
	}

	for _, story := range data.Stories {
		if story.epicLink != "" {
			story.EpicID = epicMap[story.epicLink]
		}
		jsonStr, err := json.Marshal(story)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", GetURL("stories", token), bytes.NewBuffer(jsonStr))
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
			fmt.Println("--------- *** Request Failed")
			fmt.Println("response Status:", resp.Status)
			// fmt.Println("response Headers:", resp.Header)
			fmt.Println("Request: ", story)
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
			fmt.Println("---------")
		}
	}
	return nil
}

// GetURL will get the use the REST API v1 address, the resource provided and the API token to get the URL for transactions
func GetURL(kind string, token string) string {
	return fmt.Sprintf("%s%s?token=%s", "https://api.clubhouse.io/api/v1/", kind, token)
}

// GetDataFromXMLFile will Unmarshal the XML file into the objects used by the application.
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
