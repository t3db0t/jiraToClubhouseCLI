package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
)

// JiraExport is the container of Jira Items from the XML.
type JiraExport struct {
	ElementName xml.Name   `xml:"rss"`
	Items       []JiraItem `xml:"channel>item"`
}

type JiraAssignee struct {
	Username		string	`xml:"username,attr"`
}

type JiraReporter struct {
	Username		string	`xml:"username,attr"`
}

// JiraItem is the struct for a basic item imported from the XML
type JiraItem struct {
	Assignee        JiraAssignee   `xml:"assignee"`
	CreatedAtString string   		`xml:"created"`
	Description     string   		`xml:"description"`
	Key             string   		`xml:"key"`
	Labels          []string 		`xml:"labels>label"`
	Project         string   		`xml:"project"`
	Resolution      string   		`xml:"resolution"`
	Reporter        JiraReporter  	`xml:"reporter"`
	Status          string   		`xml:"status"`
	Summary         string   		`xml:"summary"`
	Title           string   		`xml:"title"`
	Type            string   		`xml:"type"`
	Parent          string   		`xml:"parent"`

	Comments     []JiraComment     `xml:"comments>comment"`
	CustomFields []JiraCustomField `xml:"customfields>customfield"`

	epicLink string
}

//JiraCustomField is the information for custom fields. Right now the only one used is the Epic Link
type JiraCustomField struct {
	FieldName  string   `xml:"customfieldname"`
	FieldVales []string `xml:"customfieldvalues>customfieldvalue"`
}

// JiraComment is a comment from the imported XML
type JiraComment struct {
	Author          string `xml:"author,attr"`
	CreatedAtString string `xml:"created,attr"`
	Comment         string `xml:",chardata"`
	ID              string `xml:"id,attr"`
}

//GetDataForClubhouse will take the data from the XML and translate it into a format for sending to Clubhouse
func (je *JiraExport) GetDataForClubhouse(projectID int64) ClubHouseData {
	epics := []JiraItem{}
	tasks := []JiraItem{}
	stories := []JiraItem{}

	for _, item := range je.Items {
		switch item.Type {
		case "Epic":
			epics = append(epics, item)
			break
		case "Sub-task":
			tasks = append(tasks, item)
			break
		default:
			stories = append(stories, item)
			break
		}
	}

	chEpics := []ClubHouseCreateEpic{}

	for _, item := range epics {
		chEpics = append(chEpics, item.CreateEpic())
	}

	chTasks := []ClubHouseCreateTask{}
	chStories := []ClubHouseCreateStory{}

	for _, item := range tasks {
		chTasks = append(chTasks, item.CreateTask())
	}

	for _, item := range stories {
		chStories = append(chStories, item.CreateStory(projectID))
	}

	// storyMap is used to link the JiraItem's key to its index in the chStories slice. This is then used to assign subtasks properly
	storyMap := make(map[string]int)
	for i, item := range chStories {
		storyMap[item.key] = i
	}

	for _, task := range chTasks {
		chStories[storyMap[task.parent]].Tasks = append(chStories[storyMap[task.parent]].Tasks, task)
	}

	return ClubHouseData{Epics: chEpics, Stories: chStories}
}

// CreateEpic returns a ClubHouseCreateEpic from the JiraItem
func (item *JiraItem) CreateEpic() ClubHouseCreateEpic {
	return ClubHouseCreateEpic{Description: sanitize.HTML(item.Description), Name: sanitize.HTML(item.Summary), key: item.Key, CreatedAt: ParseJiraTimeStamp(item.CreatedAtString)}
}

// CreateTask returns a task if the item is a Jira Sub-task
func (item *JiraItem) CreateTask() ClubHouseCreateTask {
	return ClubHouseCreateTask{Description: sanitize.HTML(item.Summary), parent: item.Parent, Complete: false}
}

// CreateStory returns a ClubHouseCreateStory from the JiraItem
func (item *JiraItem) CreateStory(projectID int64) ClubHouseCreateStory {
	// fmt.Println("assignee: ", item.Assignee, "reporter: ", item.Reporter)
	// return ClubHouseCreateStory{}

	comments := []ClubHouseCreateComment{}
	for _, c := range item.Comments {
		comments = append(comments, c.CreateComment())
	}

	labels := []ClubHouseCreateLabel{}
	for _, label := range item.Labels {
		labels = append(labels, ClubHouseCreateLabel{Name: strings.ToLower(label)})
	}
	// Adding special label that indicates that it was imported from JIRA
	labels = append(labels, ClubHouseCreateLabel{Name: "JIRA"})

	// Overwrite supplied Project ID
	projectID = MapProject(item.Assignee.Username)

	// Map JIRA assignee to Clubhouse owner(s)
	// Leave array empty if username is unknown
	// Must use "make" function to force empty array for correct JSON marshalling
	ownerID := MapUser(item.Assignee.Username)
	var owners []string
	if ownerID != "" {
		// owners := []string{ownerID}
		owners = append(owners, ownerID)
	} else {
		owners = make([]string, 0)
	}

	// Map JIRA status to Clubhouse Workflow state
	// cases break automatically, no fallthrough by default
	var state int64 = 500000014
	switch item.Status {
	    case "Ready for Test":
	        // ready for test
	        state = 500000010
	    case "Task In Progress":
	        // in progress
	        state = 500000015
	    case "Selected for Review/Development":
	    	// selected
	    	state = 500000011
	    case "Task backlog":
	    	// backlog
	        state = 500000014
	    case "Done":
	    	// Completed
	    	state = 500000012
	    case "Verified":
	    	// Completed
	    	state = 500000012
	    case "Closed":
	    	state = 500000021
	    default:
	    	// backlog
	        state = 500000014
    }

    fmt.Printf("Creating story from %s: JIRA username: %s | Project: %d | Status: %s\n\n", item.Key, item.Assignee.Username, projectID, item.Status)

	return ClubHouseCreateStory{
		Comments:    	comments,
		CreatedAt:   	ParseJiraTimeStamp(item.CreatedAtString),
		Description: 	sanitize.HTML(item.Description),
		Labels:      	labels,
		Name:        	sanitize.HTML(item.Summary),
		ProjectID:   	projectID,
		StoryType:   	item.GetClubhouseType(),
		key:         	item.Key,
		epicLink:    	item.GetEpicLink(),
		WorkflowState:	state,
		OwnerIDs:		owners,
		RequestedBy:	MapUser(item.Reporter.Username),
	}
}

func MapUser(jiraUserName string) string {
	switch jiraUserName {
	    case "loz":
	        return "57c9ac5a-4762-46bc-bbfd-67d64bc8e7be"
	    case "ted":
	        return "570fd3d0-55a2-49f6-9352-2ddb51ef8dd1"
	    case "bruce.woodson":
	    	return "57c9a89c-d59d-4bad-aa9b-b5a0a51d8217"
	    case "carlosc":
	        return "57c9b22b-6bf4-460b-9778-764355a0bc28"
	    case "pavlo.naumenko":
	    	return "57c9c753-5ddc-41b2-93e7-9f4474d82380"
	    case "yuchao.chen":
	    	return "57c9b277-47b1-4277-a92a-364e07871c46"
	    case "dmitriy":
	    	return ""
	    case "yuri.cantor":
	    	return "57c9b19b-917a-457d-a4be-7a1d12990305"
	    case "hyoung.kim":
	    	return "57c9ab13-9a97-44a9-bfbe-2971cabcab9f"
	    case "jamesb":
	    	return "57c9c7ae-4f1b-4685-826c-26056923460e"
    	case "jason.oliver":
	    	return "57680d04-f3f5-4dcc-bcd7-06cd79452398"
	    case "mikhail":
	    	return "576ad70f-0df0-4649-bbb7-aa48cb024c2f"
	    case "britton.sparks":
	    	return "578d49c3-b68f-4ca6-98bd-743545e8a8e9"
	    case "thomast":
	    	return "577d43f6-e069-417d-ba40-5704ba82dc7c"
	    case "vadym.lipinsky":
	    	return "57c9a83f-f360-4f01-a667-5eb632394ff1"
	    case "zach.mihalko":
	    	return ""
	    case "gregmihalko":
	    	return ""
	    case "jeremy.leon":
	    	return ""
	    default:
	    	fmt.Println("[MapUser] JIRA Assignee not found: ", jiraUserName)
	    	return ""
    }
}

/* Overwrite supplied Project ID
	   if assigned to loz, Frontend-Import
	   if item.Summary starts with "FE: ", Frontend-Import
	   if item.Summary contains "Touch", Touch
	   etc

	- Frontend: 5
	- Backend: 6
	- Gateway: 7
	- Touch: 19
	- Android App: 277
	- New Products (Ted): 81
	- QA: 295
	- QA Automation: 287
	- iOS App: 246
	- Infrastructure: 298
	- Misc: 299
	*/

func MapProject(jiraUserName string) int64 {
	switch jiraUserName {
	    case "loz":
	    	return 5
	    case "ted":
	    	return 81
	    case "bruce.woodson":
	    	return 6
	    case "carlosc":
	        return 7
	    case "pavlo.naumenko":
	    	return 246
	    case "yuchao.chen":
	    	return 5
	    case "dmitriy":
	    	return 6
	    case "yuri.cantor":
	    	return 298
	    case "hyoung.kim":
	    	return 19
	    case "jamesb":
	    	return 298
    	case "jason.oliver":
	    	return 81
	    case "mikhail":
	    	return 7
	    case "britton.sparks":
	    	return 295
	    case "thomast":
	    	return 295
	    case "vadym.lipinsky":
	    	return 287
	    case "zach.mihalko":
	    	return 318
	    case "gregmihalko":
	    	return 318
	    case "jeremy.leon":
	    	return 287
	    default:
	    	fmt.Println("[MapProject] JIRA Assignee not found: ", jiraUserName)
	    	return 299
    }
}


// CreateComment takes the JiraItem's comment data and returns a ClubHouseCreateComment
func (comment *JiraComment) CreateComment() ClubHouseCreateComment {
	return ClubHouseCreateComment{
		Text:		sanitize.HTML(comment.Comment),
		CreatedAt:	ParseJiraTimeStamp(comment.CreatedAtString),
		Author: 	MapUser(comment.Author),
	}
}

// GetEpicLink returns the Epic Link of a Jira Item.
func (item *JiraItem) GetEpicLink() string {
	for _, cf := range item.CustomFields {
		if cf.FieldName == "Epic Link" {
			return cf.FieldVales[0]
		}
	}
	return ""
}

// GetClubhouseType determines type based on if the Jira item is a bug or not.
func (item *JiraItem) GetClubhouseType() string {
	if item.Type == "Bug" {
		return "bug"
	}
	return "feature"
}

// ParseJiraTimeStamp parses the format in the XML using Go's magical timestamp.
func ParseJiraTimeStamp(dateString string) time.Time {
	format := "Mon, 2 Jan 2006 15:04:05 -0700"
	t, err := time.Parse(format, dateString)
	if err != nil {
		return time.Now()
	}
	return t
}
